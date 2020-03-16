package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
)

type TelegramBody struct {
	ChatId string `json:"chat_id"`
	Text   string `json:"text"`
}

func getCostReport(startDate string, endDate string) string {
	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		log.Fatal("Unable to load SDK config", err.Error())
	}

	timePeriod := costexplorer.DateInterval{
		Start: aws.String(startDate),
		End:   aws.String(endDate),
	}
	ce := costexplorer.New(cfg)
	payload := costexplorer.GetCostAndUsageInput{
		Granularity: costexplorer.GranularityMonthly,
		TimePeriod:  &timePeriod,
		Metrics:     []string{"UnblendedCost"},
		GroupBy: []costexplorer.GroupDefinition{
			{
				Key:  aws.String("SERVICE"),
				Type: costexplorer.GroupDefinitionTypeDimension,
			},
		},
	}

	req := ce.GetCostAndUsageRequest(&payload)
	resp, err := req.Send(context.Background())
	if err != nil {
		log.Fatalf("Unable to find cost report\nError: %s", err.Error())
	}

	var report string
	total := 0.0
	for _, result := range resp.ResultsByTime {
		for _, v := range result.Groups {
			service := strings.Join(v.Keys[:], ", ")
			amount := *v.Metrics["UnblendedCost"].Amount
			pAmount, _ := strconv.ParseFloat(amount, 64)
			total += pAmount
			report += fmt.Sprintf("%s\n$%s\n\n", service, amount)
		}
	}

	report += fmt.Sprintf("Total\n$%f\n", total)
	return report
}

func sendMessage(report string, token string, chatId string) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	payload := TelegramBody{
		ChatId: chatId,
		Text:   report,
	}
	pbytes, _ := json.Marshal(payload)
	buff := bytes.NewBuffer(pbytes)
	_, err := http.Post(url, "application/json", buff)
	if err != nil {
		log.Fatal("message could not be sent. " + err.Error())
	}
}

func Handler() {
	t := time.Now()
	startDate := fmt.Sprintf("%d-%02d-%02d", t.Year(), t.Month(), 1)
	endDate := fmt.Sprintf("%d-%02d-%02d", t.Year(), t.Month(), t.Day())

	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatId := os.Getenv("TELEGRAM_BOT_CHAT_ID")

	if token == "" || chatId == "" {
		log.Fatal("Unable to load Telegram config")
	}

	report := getCostReport(startDate, endDate)
	sendMessage(report, token, chatId)
}

func main() {
	lambda.Start(Handler)
}

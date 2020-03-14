.PHONY: build clean deploy

build:
	env GOOS=linux go build -ldflags="-s -w" -o bin/report functions/report.go

clean:
	rm -rf ./bin

deploy: clean build
	sls deploy --stage production

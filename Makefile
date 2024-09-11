build:
	GOARCH=amd64 GOOS=linux go build -o bootstrap main.go
	zip deployment.zip bootstrap
deploy: build
	serverless deploy --stage dev
clean:
	rm -rf ./bin ./vendor Gopkg.lock ./serverless

package main

import (
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	fmt.Println("Event:", request)
	cloudfrontOrigin := os.Getenv("CloudfrontOrigin")
	headers := map[string]string{
		"Access-Control-Allow-Origin":   cloudfrontOrigin,
		"Access-Control-Allow-Headers:": "*",
	}

	return events.APIGatewayProxyResponse{
		Headers:    headers,
		Body:       os.Getenv("ApiKey"),
		StatusCode: 200,
	}, nil
}

func main() {
	lambda.Start(handler)
}

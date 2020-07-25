package main

import (
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/aws/aws-sdk-go/aws/session"

	"github.com/aws/aws-sdk-go/service/ssm"
)

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	cloudfrontOrigin := os.Getenv("CloudfrontOrigin")
	keyName := os.Getenv("TimerKeyName")
	fmt.Println("keyName:", keyName)
	headers := map[string]string{
		"Access-Control-Allow-Origin":   cloudfrontOrigin,
		"Access-Control-Allow-Headers:": "*",
	}
	fmt.Println("Starting session...")
	svc := ssm.New(session.New())
	input := &ssm.GetParameterInput{Name: &keyName}
	response, err := svc.GetParameter(input)
	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    headers,
			Body:       err.Error(),
			StatusCode: 400,
		}, nil
	}

	return events.APIGatewayProxyResponse{
		Headers:    headers,
		Body:       *response.Parameter.Value,
		StatusCode: 200,
	}, nil
}

func main() {
	lambda.Start(handler)
}

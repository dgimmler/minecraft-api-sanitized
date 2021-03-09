package main

import (
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
)

// Creates (or updates if already exists) parameter store parameter with status
// of "started" to indicate that the server is running. Returns success/failure
// of function
func markAsStarted() error {
	fmt.Println("Starting session...")
	sess := session.New()

	// set properties
	keyName := os.Getenv("ServerStatusKeyName")
	fmt.Println("ServerStatusKeyName:", keyName)
	value := "started"
	paramType := "String"
	desc := "Status of minecraft server. Status reflects specifically the status of the minecraft service ON the server, not the server itself."
	overwrite := true // overwrite if it already exists
	input := &ssm.PutParameterInput{
		Description: &desc,
		Name:        &keyName,
		Overwrite:   &overwrite,
		Value:       &value,
		Type:        &paramType,
	}

	// create/upudate parameter
	svc := ssm.New(sess)
	_, err := svc.PutParameter(input)
	if err != nil {
		return err
	}

	fmt.Println("Marked server as stopped")
	return nil
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	fmt.Println("Event:", request)
	cloudfrontOrigin := os.Getenv("CloudfrontOrigin")
	headers := map[string]string{
		"Access-Control-Allow-Origin":   cloudfrontOrigin,
		"Access-Control-Allow-Headers:": "*",
	}

	err := markAsStarted()
	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    headers,
			Body:       err.Error(),
			StatusCode: 400,
		}, nil
	}

	return events.APIGatewayProxyResponse{
		Headers:    headers,
		Body:       "success",
		StatusCode: 200,
	}, nil
}

func main() {
	lambda.Start(handler)
}

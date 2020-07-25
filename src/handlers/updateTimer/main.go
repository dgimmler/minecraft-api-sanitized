package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/aws/aws-sdk-go/aws/session"

	"github.com/aws/aws-sdk-go/service/ssm"
)

// Creates (or updates if already exists) parameter store parameter with unix
// time stamp 2 hours from now to act as timer for automatically shutting down
// server. Returns success/failure of function
func updateTimer(sess *session.Session, value string) error {
	// set properties
	keyName := os.Getenv("TimerKeyName")
	fmt.Println("TimerKeyName:", keyName)
	paramType := "String"
	desc := "Unix timestamp for auto-shutting down minecraft server"
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

	fmt.Println("Set stop time")
	return nil
}

// Body to marshal json request into
type Body struct {
	Value string `json:"value"`
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	cloudfrontOrigin := os.Getenv("CloudfrontOrigin")
	headers := map[string]string{
		"Access-Control-Allow-Origin":   cloudfrontOrigin,
		"Access-Control-Allow-Headers:": "*",
	}
	
	// parse request body
	var body Body
	err := json.Unmarshal([]byte(request.Body), &body)
	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    headers,
			Body:       err.Error(),
			StatusCode: 400,
		}, nil
	}
	fmt.Println("new value:", body.Value)

	// start aws session
	fmt.Println("Starting session...")
	sess := session.New()

	// set stop time as unix timestamp parameter in parameter store
	err = updateTimer(sess, body.Value)
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

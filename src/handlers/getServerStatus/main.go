package main

import (
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ssm"
)

// getServiceStatus returns the status of the actual minecraft service ON the
// server
func getServiceStatus(sess *session.Session) (string, error) {
	keyName := os.Getenv("ServerStatusKeyName")
	svc := ssm.New(sess)
	input := &ssm.GetParameterInput{Name: &keyName}
	_, err := svc.GetParameter(input)
	if err != nil {
		return "", err
	}
	return "running", nil
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	cloudfrontOrigin := os.Getenv("CloudfrontOrigin")
	instanceID := os.Getenv("ServerId")
	headers := map[string]string{
		"Access-Control-Allow-Origin":   cloudfrontOrigin,
		"Access-Control-Allow-Headers:": "*",
	}
	fmt.Println("Starting session...")
	sess := session.New()
	svc := ec2.New(sess)
	fmt.Println("Retrieving instance", instanceID, "...")
	input := &ec2.DescribeInstanceStatusInput{
		InstanceIds: []*string{
			aws.String(instanceID),
		},
		IncludeAllInstances: aws.Bool(true),
	}
	result, err := svc.DescribeInstanceStatus(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				msg := fmt.Sprintf("error retrieving instance status: %s", aerr.Error())
				fmt.Println(msg)
				return events.APIGatewayProxyResponse{
					Headers:    headers,
					Body:       msg,
					StatusCode: 400,
				}, nil
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			msg := fmt.Sprintf("error retrieving instance status: %s", aerr.Error())
			fmt.Println(msg)
			return events.APIGatewayProxyResponse{
				Headers:    headers,
				Body:       msg,
				StatusCode: 400,
			}, nil
		}
	}
	if len(result.InstanceStatuses) < 1 {
		msg := fmt.Sprintf("Could not find instance with ID %s", instanceID)
		fmt.Println(msg)
		return events.APIGatewayProxyResponse{
			Headers:    headers,
			Body:       msg,
			StatusCode: 200,
		}, nil
	}

	// get state of the server itself
	fmt.Println("status:", result.InstanceStatuses)
	InstanceState := result.InstanceStatuses[0].InstanceState.Name

	// if the server is on, get state of the minecraft service ON the server
	if *InstanceState == "running" {
		status, err := getServiceStatus(sess)
		if err != nil {
			// err occurs because parameter does not yet exist, indicating it's
			// pending
			return events.APIGatewayProxyResponse{
				Headers:    headers,
				Body:       "pending",
				StatusCode: 200,
			}, nil
		}

		fmt.Println("service status:", status)
		return events.APIGatewayProxyResponse{
			Headers:    headers,
			Body:       status,
			StatusCode: 200,
		}, nil
	}

	// otherwise just return the current status
	fmt.Println("instance state:", *InstanceState)
	return events.APIGatewayProxyResponse{
		Headers:    headers,
		Body:       *InstanceState,
		StatusCode: 200,
	}, nil
}

func main() {
	lambda.Start(handler)
}

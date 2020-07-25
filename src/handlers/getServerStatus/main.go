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
)

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	cloudfrontOrigin := os.Getenv("CloudfrontOrigin")
	instanceID := os.Getenv("ServerId")
	headers := map[string]string{
		"Access-Control-Allow-Origin":   cloudfrontOrigin,
		"Access-Control-Allow-Headers:": "*",
	}
	fmt.Println("Starting session...")
	svc := ec2.New(session.New())
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
	fmt.Println("status:", result.InstanceStatuses)
	InstanceState := result.InstanceStatuses[0].InstanceState.Name

	return events.APIGatewayProxyResponse{
		Headers:    headers,
		Body:       *InstanceState,
		StatusCode: 200,
	}, nil
}

func main() {
	lambda.Start(handler)
}

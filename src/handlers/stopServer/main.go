package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/aws/aws-sdk-go/service/cloudwatchevents"
	"github.com/aws/aws-sdk-go/service/ssm"
)

// Event is only used to verify if source was scheduled cloudwatch rule. We only
// need to parse a single value to validate this source.
type Event struct {
	Source string `json:"source"`
}

// delete parameter store value
func deleteParameter(sess *session.Session) error {
	fmt.Println("deleting parameter...")
	keyName := os.Getenv("TimerKeyName")
	svc := ssm.New(sess)
	input := &ssm.DeleteParameterInput{Name: &keyName}
	_, err := svc.DeleteParameter(input)
	if err != nil {
		return err
	}
	return nil
}

// removes lambda target from rule so that it can be deleted
func removeTarget(sess *session.Session) error {
	fmt.Println("removing target...")
	svc := cloudwatchevents.New(sess)
	id := "stopServerScheduledStopLambdaTarget"
	name := os.Getenv("CloudwatchRuleName")
	input := &cloudwatchevents.RemoveTargetsInput{
		Ids:  []*string{&id},
		Rule: &name,
	}
	_, err := svc.RemoveTargets(input)
	if err != nil {
		return err
	}

	return nil
}

// delete event rule
func deleteRule(sess *session.Session) error {
	fmt.Println("deleting rule...")
	err := removeTarget(sess)
	if err != nil {
		return err
	}
	svc := cloudwatchevents.New(sess)
	name := os.Getenv("CloudwatchRuleName")
	input := &cloudwatchevents.DeleteRuleInput{Name: &name}
	_, err = svc.DeleteRule(input)
	if err != nil {
		return err
	}

	return nil
}

// get scheduled stop time from parameter store
func getServerTimer(sess *session.Session) (string, error) {
	keyName := os.Getenv("TimerKeyName")
	fmt.Println("keyName:", keyName)
	svc := ssm.New(sess)
	input := &ssm.GetParameterInput{Name: &keyName}
	response, err := svc.GetParameter(input)
	if err != nil {
		return "", err
	}
	return *response.Parameter.Value, nil
}

// return true if current time is past scheduled stop time
func isScheduledToStop(sess *session.Session) (bool, error) {
	fmt.Println("Scheduled to stop, checking stop time...")
	value, err := getServerTimer(sess)
	if err != nil {
		return false, err
	}

	stopTime, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return false, err
	}
	now := int64(time.Now().Unix())
	if now >= stopTime {
		return true, nil
	}

	return false, nil
}

func handler(request Event) (events.APIGatewayProxyResponse, error) {
	fmt.Println("Event:", request)
	cloudfrontOrigin := os.Getenv("CloudfrontOrigin")
	instanceID := os.Getenv("ServerId")
	headers := map[string]string{
		"Access-Control-Allow-Origin":   cloudfrontOrigin,
		"Access-Control-Allow-Headers:": "*",
	}
	fmt.Println("Starting session...")
	sess := session.New()

	// if lambda was triggered by scheduled event, first check to see if server
	// is scheduuled to stop yet
	if request.Source == "aws.events" {
		shouldStop, err := isScheduledToStop(sess)
		if err != nil {
			return events.APIGatewayProxyResponse{
				Headers:    headers,
				Body:       err.Error(),
				StatusCode: 400,
			}, nil
		}

		if !shouldStop {
			return events.APIGatewayProxyResponse{
				Headers:    headers,
				Body:       "Not yet scheduled to stop",
				StatusCode: 200,
			}, nil
		}
	}

	fmt.Println("Stopping instance", instanceID, "...")
	svc := ec2.New(sess)
	input := &ec2.StopInstancesInput{
		InstanceIds: []*string{
			aws.String(instanceID),
		},
	}
	result, err := svc.StopInstances(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println("error retrieving instance status:", aerr.Error())
				return events.APIGatewayProxyResponse{
					Headers:    headers,
					Body:       aerr.Error(),
					StatusCode: 400,
				}, nil
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println("error retrieving instance status:", err.Error())
			return events.APIGatewayProxyResponse{
				Headers:    headers,
				Body:       err.Error(),
				StatusCode: 400,
			}, nil
		}
	}
	if len(result.StoppingInstances) < 1 {
		msg := fmt.Sprintf("Could not find instance with ID %s", instanceID)
		fmt.Println(msg)
		return events.APIGatewayProxyResponse{
			Headers:    headers,
			Body:       msg,
			StatusCode: 200,
		}, nil
	}
	fmt.Println("status:", result.StoppingInstances)

	// if server is successfully stopped, delete the event rule
	err = deleteRule(sess)
	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    headers,
			Body:       err.Error(),
			StatusCode: 400,
		}, nil
	}

	// then delete parameters tore value, just to clean everything up
	err = deleteParameter(sess)
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

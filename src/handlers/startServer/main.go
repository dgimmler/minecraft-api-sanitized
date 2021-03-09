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

	"github.com/aws/aws-sdk-go/service/cloudwatchevents"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ssm"
)

// Creates (or updates if already exists) parameter store parameter with status
// of "starting" to indicate that the server is running. Returns success/failure
// of function
func markAsStarting(sess *session.Session) error {
	// set properties
	keyName := os.Getenv("ServerStatusKeyName")
	fmt.Println("ServerStatusKeyName:", keyName)
	value := "starting"
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

func scheduleStop(sess *session.Session) error {
	fmt.Println("scheduling auto-stopper")
	svc := cloudwatchevents.New(sess)

	// we must first create the schedule, then set the target (2 calls)
	description := "Checks stop time for minecraft server every 30 minutes and stops server if past stop time"
	name := os.Getenv("CloudwatchRuleName")
	schedule := "rate(1 minute)"
	state := "ENABLED"
	ruleInput := &cloudwatchevents.PutRuleInput{
		Description:        &description,
		Name:               &name,
		ScheduleExpression: &schedule,
		State:              &state,
	}
	_, err := svc.PutRule(ruleInput)
	if err != nil {
		return err
	}

	// add stopServer lambda as rule target
	id := "stopServerScheduledStopLambdaTarget"
	arn := os.Getenv("StopServerArn")
	fmt.Println("arn:", arn)
	targetInput := &cloudwatchevents.PutTargetsInput{
		Rule: &name,
		Targets: []*cloudwatchevents.Target{&cloudwatchevents.Target{
			Id:  &id,
			Arn: &arn,
		}},
	}
	_, err = svc.PutTargets(targetInput)
	if err != nil {
		return err
	}

	fmt.Println("scheduled stopTime")
	return nil
}

// Creates (or updates if already exists) parameter store parameter with unix
// time stamp 2 hours from now to act as timer for automatically shutting down
// server. Returns success/failure of function
func startTimer(sess *session.Session) error {
	// set properties
	keyName := os.Getenv("TimerKeyName")
	fmt.Println("TimerKeyName:", keyName)
	paramType := "String"
	desc := "Unix timestamp for auto-shutting down minecraft server"
	overwrite := true                                                       // overwrite if it already exists
	stopTime := strconv.FormatInt(int64(time.Now().Unix())+int64(7140), 10) // 1:59 hours from now (give it one min buffer)
	fmt.Println("stopTime:", stopTime)
	input := &ssm.PutParameterInput{
		Description: &desc,
		Name:        &keyName,
		Overwrite:   &overwrite,
		Value:       &stopTime,
		Type:        &paramType,
	}

	// create/upudate parameter
	svc := ssm.New(sess)
	_, err := svc.PutParameter(input)
	if err != nil {
		return err
	}

	fmt.Println("Set stop time")
	return scheduleStop(sess)
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// TODO add funciton to create cloudwatch schedule. Make sure it happens
	// AFTER the parameter is created. Should schedule for every 30 min
	cloudfrontOrigin := os.Getenv("CloudfrontOrigin")
	instanceID := os.Getenv("ServerId")
	headers := map[string]string{
		"Access-Control-Allow-Origin":   cloudfrontOrigin,
		"Access-Control-Allow-Headers:": "*",
	}
	fmt.Println("Starting session...")
	sess := session.New()
	svc := ec2.New(sess)
	fmt.Println("Starting instance", instanceID, "...")
	input := &ec2.StartInstancesInput{
		InstanceIds: []*string{
			aws.String(instanceID),
		},
	}
	result, err := svc.StartInstances(input)
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
	fmt.Println("status:", result.StartingInstances)
	if len(result.StartingInstances) < 1 {
		msg := fmt.Sprintf("Could not find instance with ID %s", instanceID)
		fmt.Println(msg)
		return events.APIGatewayProxyResponse{
			Headers:    headers,
			Body:       msg,
			StatusCode: 200,
		}, nil
	}
	fmt.Println("status:", result.StartingInstances)

	// set stop time as unix timestamp parameter in parameter store
	err = startTimer(sess)
	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    headers,
			Body:       err.Error(),
			StatusCode: 400,
		}, nil
	}

	// then create or update schedule to trigger lambda every 30 minutes
	err = scheduleStop(sess)
	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    headers,
			Body:       err.Error(),
			StatusCode: 400,
		}, nil
	}

	// finally, create or update parameter store value to indicate server is
	// booting up. This will be updated as "started" once the minecraft service
	// itself is actually up and running ON the server
	// (commented out for now, but leaving in in case we want it back easily)
	// err = markAsStarting(sess)
	// if err != nil {
	// 	return events.APIGatewayProxyResponse{
	// 		Headers:    headers,
	// 		Body:       err.Error(),
	// 		StatusCode: 400,
	// 	}, nil
	// }

	return events.APIGatewayProxyResponse{
		Headers:    headers,
		Body:       "success",
		StatusCode: 200,
	}, nil
}

func main() {
	lambda.Start(handler)
}

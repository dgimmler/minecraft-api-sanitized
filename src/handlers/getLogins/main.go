package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

// NewClient creates and returns new dynamodb client
func NewClient() *dynamodb.DynamoDB {
	region := os.Getenv("Region")
	fmt.Println("region:", region)
	config := &aws.Config{Region: aws.String(region)}
	sess := session.Must(session.NewSession(config))
	client := dynamodb.New(sess)
	fmt.Println("[NewClient]", "Created client")
	return client
}

// Query takes in list of usernames
type Query struct {
	Usernames []string `json:"Usernames"`
}

// NewQuery creates and returns new DynamoDbItem
func NewQuery(body string) (*Query, error) {
	fmt.Println("[NewQuery]", "body:", body)
	var q Query
	if body == "" {
		fmt.Println("[NewQuery] No username filter provided")
		q.Usernames = []string{"*"} // set single username of "*" to indicate no filter
		return &q, nil
	}
	err := json.Unmarshal([]byte(body), &q)
	if err != nil {
		fmt.Println("[NewQuery]", err)
		return nil, err
	}
	fmt.Println("[NewQuery]", "Created new Query")
	fmt.Println("[NewQuery]", q)
	return &q, nil
}

// DynamoDbItem is a struct for capturing returned dynamodb item attributes
type DynamoDbItem struct {
	PK         string `json:"Username" dynamodbav:"PK"`
	SK         string `json:"Version" dynamodbav:"SK,omitempty"`
	LoginTime  int32  `json:"LoginTime" dynamodbav:"LoginTime"`
	LogoutTime int32  `json:"LogoutTime" dynamodbav:"LogoutTime,omitempty"`
}

// getUserLogins queries for the logins of a specific user
func getUserLogins(tableName string, client *dynamodb.DynamoDB, q *Query) ([]DynamoDbItem, error) {
	var logins []DynamoDbItem
	input := &dynamodb.QueryInput{
		ScanIndexForward: aws.Bool(false),
		TableName:        aws.String(tableName),
	}
	for _, s := range q.Usernames {
		if s != "*" {
			// query username index of specific username
			input.ExpressionAttributeValues = map[string]*dynamodb.AttributeValue{
				":u": {
					S: aws.String(s),
				},
			}
			input.KeyConditionExpression = aws.String("PK = :u")
			input.IndexName = aws.String("Username")
		} else {
			// query version indiex to just get all users (for current version)
			input.ExpressionAttributeValues = map[string]*dynamodb.AttributeValue{
				":v": {
					S: aws.String("v1"), // TODO set version as env var
				},
			}
			input.KeyConditionExpression = aws.String("SK = :v")
			input.IndexName = aws.String("Version")
		}

		fmt.Println("input:", input)
		result, err := client.Query(input)
		if err != nil {
			fmt.Println("[getUserLogins]", err)
			return logins, err
		}
		fmt.Println("result:", result)

		// parse readmes
		dbi := []DynamoDbItem{}
		err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &dbi)
		if err != nil {
			return logins, err
		}
		fmt.Println("logins:", dbi)
		logins = append(logins, dbi...)
	}

	return logins, nil
}

// Handler is main entry point to lambda function
func Handler(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	tableName := os.Getenv("UserLoginTableName")
	origin := os.Getenv("CloudfrontOrigin")
	fmt.Println("[Handler]", "Searching table ", tableName)
	fmt.Println("[Handler]", "Cloudfront origin ", origin)
	headers := map[string]string{
		"Access-Control-Allow-Origin":      origin,
		"Access-Control-Allow-Credentials": "true",
		"Access-Control-Allow-Methods":     "OPTIONS,POST",
		"Access-Control-Allow-Headers":     "*",
	}

	q, err := NewQuery(event.Body)
	if err != nil {
		// error handling for NewAttributeValue above (needed headers for
		// response)
		fmt.Println("[Handler]", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       err.Error(),
			Headers:    headers,
		}, nil
	}

	client := NewClient()
	logins, err := getUserLogins(tableName, client, q)
	if err != nil {
		fmt.Println("[Handler]", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       err.Error(),
			Headers:    headers,
		}, nil
	}

	// get stringified json to return
	fmt.Println("logins:", logins)
	loginsJSON, err := json.Marshal(logins)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       err.Error(),
			Headers:    headers,
		}, nil
	}
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       string(loginsJSON),
		Headers:    headers,
	}, nil
}
func main() {
	lambda.Start(Handler)
}

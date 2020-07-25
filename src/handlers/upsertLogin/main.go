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

// DynamoDbItem is a struct for capturing passed dynamodb item attributes
type DynamoDbItem struct {
	PK         string `json:"Username" dynamodbav:"PK"`
	SK         string `json:"Version" dynamodbav:"SK"`
	LoginTime  int32  `json:"LoginTime" dynamodbav:"LoginTime"`
	LogoutTime int32  `json:"LogoutTime" dynamodbav:"LogoutTime,omitempty"`
}

// NewDynamoDbItem Creates and returns new DynamoDbItem
func NewDynamoDbItem(body string) (*DynamoDbItem, error) {
	fmt.Println("[NewDynamoDbItem]", "body:", body)
	var b DynamoDbItem
	err := json.Unmarshal([]byte(body), &b)
	if err != nil {
		fmt.Println("[NewDynamoDbItem]", err)
		return nil, err
	}

	fmt.Println("[NewDynamoDbItem]", "Created new DynamoDbItem")
	fmt.Println("[NewDynamoDbItem]", b)
	return &b, nil
}

// NewClient creates and returns new dynamodb client
func NewClient() *dynamodb.DynamoDB {
	region := os.Getenv("Region")
	fmt.Println("Region:", region)
	config := &aws.Config{Region: aws.String(region)}
	sess := session.Must(session.NewSession(config))
	client := dynamodb.New(sess)
	fmt.Println("[NewClient]", "Created client")
	return client
}

// NewAttributeValue creates and returns new dynamodb.AttributeValue. This is
// the object type containing the item data exptected by the dynamodb API
func NewAttributeValue(body string) (map[string]*dynamodb.AttributeValue, error) {
	b, err := NewDynamoDbItem(body)
	if err != nil {
		fmt.Println("[NewAttributeValue]", err)
		return nil, err
	}
	fmt.Println("[NewAttributeValue]", "Created AttributeValue")
	item, err := dynamodbattribute.MarshalMap(b)
	if err != nil {
		fmt.Println("[NewAttributeValue]", err)
		return nil, err
	}
	fmt.Println("[NewAttributeValue]", "Created item")
	fmt.Println("[NewAttributeValue", item)
	return item, nil
}

// Handler is the main function for lambda
func Handler(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// get tablename, origin and item attributes
	tableName := os.Getenv("UserLoginTableName")
	origin := os.Getenv("CloudfrontOrigin")
	attrVal, err := NewAttributeValue(event.Body)
	fmt.Println("[Handler]", "Updating table ", tableName)
	fmt.Println("[Handler]", "Cloudfront origin ", origin)
	headers := map[string]string{
		"Access-Control-Allow-Origin":      origin,
		"Access-Control-Allow-Credentials": "true",
		"Access-Control-Allow-Methods":     "OPTIONS,POST",
		"Access-Control-Allow-Headers":     "*",
	}
	if err != nil {
		// error handling for NewAttributeValue above (needed headers for
		// response)
		fmt.Println("[Handler] [NewAttributeValue]", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       err.Error(),
			Headers:    headers,
		}, nil
	}
	fmt.Println("[Handler]", "Rettrieved attrVal")
	input := &dynamodb.PutItemInput{
		Item:                   attrVal,
		ReturnConsumedCapacity: aws.String("TOTAL"),
		TableName:              aws.String(tableName),
	}
	fmt.Println("[Handler]", "Created input")
	fmt.Println("[Handler]", input)

	// create item
	client := NewClient()
	result, err := client.PutItem(input)
	if err != nil {
		fmt.Println("[Handler]", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       err.Error(),
			Headers:    headers,
		}, nil
	}
	fmt.Println("[Handler]", "Called PutItem")

	// get stringified json to return
	fmt.Println("logins:", result)
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       err.Error(),
			Headers:    headers,
		}, nil
	}

	// return result
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       string(resultJSON),
		Headers:    headers,
	}, nil
}

func main() {
	lambda.Start(Handler)
}

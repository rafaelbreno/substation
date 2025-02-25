package dynamodb

import (
	"fmt"
	"os"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-xray-sdk-go/xray"
)

// New returns a configured DynamoDB client.
func New() *dynamodb.DynamoDB {
	conf := aws.NewConfig()

	// provides forward compatibility for the Go SDK to support env var configuration settings
	// https://github.com/aws/aws-sdk-go/issues/4207
	max, found := os.LookupEnv("AWS_MAX_ATTEMPTS")
	if found {
		m, err := strconv.Atoi(max)
		if err != nil {
			panic(err)
		}

		conf = conf.WithMaxRetries(m)
	}

	c := dynamodb.New(
		session.Must(session.NewSession()),
		conf,
	)

	if _, ok := os.LookupEnv("AWS_XRAY_DAEMON_ADDRESS"); ok {
		xray.AWS(c.Client)
	}

	return c
}

// API wraps the DynamoDB API interface.
type API struct {
	Client dynamodbiface.DynamoDBAPI
}

// Setup creates a new DynamoDB client.
func (a *API) Setup() {
	a.Client = New()
}

// IsEnabled returns true if the client is enabled and ready for use.
func (a *API) IsEnabled() bool {
	return a.Client != nil
}

// PutItem is a convenience wrapper for putting items into a DynamoDB table.
func (a *API) PutItem(ctx aws.Context, table string, item map[string]*dynamodb.AttributeValue) (resp *dynamodb.PutItemOutput, err error) {
	resp, err = a.Client.PutItemWithContext(
		ctx,
		&dynamodb.PutItemInput{
			TableName: aws.String(table),
			Item:      item,
		})

	if err != nil {
		return nil, fmt.Errorf("putitem table %s: %v", table, err)
	}

	return resp, nil
}

/*
Query is a convenience wrapper for querying a DynamoDB table. The paritition and sort keys are always referenced in the key condition expression as ":pk" and ":sk". Refer to the DynamoDB documentation for the Query operation's request syntax and key condition expression patterns:

- https://docs.aws.amazon.com/amazondynamodb/latest/APIReference/API_Query.html#API_Query_RequestSyntax

- https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/Query.html#Query.KeyConditionExpressions
*/
func (a *API) Query(ctx aws.Context, table, partitionKey, sortKey, keyConditionExpression string, limit int64, scanIndexForward bool) (resp *dynamodb.QueryOutput, err error) {
	expression := make(map[string]*dynamodb.AttributeValue)
	expression[":pk"] = &dynamodb.AttributeValue{
		S: aws.String(partitionKey),
	}

	if sortKey != "" {
		expression[":sk"] = &dynamodb.AttributeValue{
			S: aws.String(sortKey),
		}
	}

	resp, err = a.Client.QueryWithContext(
		ctx,
		&dynamodb.QueryInput{
			TableName:                 aws.String(table),
			KeyConditionExpression:    aws.String(keyConditionExpression),
			ExpressionAttributeValues: expression,
			Limit:                     aws.Int64(limit),
			ScanIndexForward:          aws.Bool(scanIndexForward),
		})
	if err != nil {
		return nil, fmt.Errorf("query table %s key condition expression %s: %v", table, keyConditionExpression, err)
	}

	return resp, nil
}

// GetItem is a convenience wrapper for getting items into a DynamoDB table.
func (a *API) GetItem(ctx aws.Context, table string, attributes map[string]interface{}) (resp *dynamodb.GetItemOutput, err error) {
	attr, err := dynamodbattribute.MarshalMap(attributes)
	if err != nil {
		return nil, fmt.Errorf("get_item: table %s: %v", table, err)
	}

	resp, err = a.Client.GetItemWithContext(
		ctx,
		&dynamodb.GetItemInput{
			TableName: aws.String(table),
			Key:       attr,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("get_item: table %s: %v", table, err)
	}

	return resp, nil
}

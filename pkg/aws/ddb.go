package aws

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

type ErrTableExists struct{}

func (e ErrTableExists) Error() string {
	return "table already exists"
}

func NewDBConn() *dynamodb.DynamoDB {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	return dynamodb.New(sess)
}

func ListAllTables(db *dynamodb.DynamoDB) ([]string, error) {
	input := &dynamodb.ListTablesInput{}

	result, err := db.ListTables(input)
	if err != nil {
		var aerr awserr.Error
		if errors.As(err, &aerr) {
			return nil, fmt.Errorf("aws error: %s", aerr.Error())
		} else {
			return nil, fmt.Errorf("listing tables, non aws error: %w", err)
		}
	}

	var tableNames []string
	for _, n := range result.TableNames {
		tableNames = append(tableNames, *n)
	}

	return tableNames, nil
}

func CreateTableIfNotExist(db *dynamodb.DynamoDB, input *dynamodb.CreateTableInput) error {
	tables, err := ListAllTables(db)
	if err != nil {
		return fmt.Errorf("listing tables: %w", err)
	}

	for _, name := range tables {
		if name == *input.TableName {
			return ErrTableExists{}
		}
	}

	if _, err := db.CreateTable(input); err != nil {
		return fmt.Errorf("creating table: %w", err)
	}

	return nil
}

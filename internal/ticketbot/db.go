package ticketbot

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"strconv"
	"tctg-automation/pkg/amz"
)

const (
	tableName = "ticketbot-boards"
)

func (s *Server) listBoards() ([]boardSetting, error) {
	if err := s.createOrAddBoardTable(); err != nil {
		return nil, fmt.Errorf("ensuring board table exists: %w", err)
	}

	params := &dynamodb.ScanInput{
		TableName: aws.String(tableName),
	}
	result, err := s.db.Scan(params)
	if err != nil {
		return nil, fmt.Errorf("scanning items: %w", err)
	}

	var boards []boardSetting
	for _, i := range result.Items {
		idStr := i["BoardId"]
		if idStr == nil || idStr.N == nil {
			return nil, fmt.Errorf("board item missing BoardId")
		}

		boardID, err := strconv.Atoi(*idStr.N)
		if err != nil {
			return nil, fmt.Errorf("parsing BoardId: %w", err)
		}

		board := boardSetting{
			BoardID:     boardID,
			BoardName:   *i["BoardName"].S,
			WebexRoomID: *i["WebexRoomId"].S,
		}
		boards = append(boards, board)
	}

	return boards, nil
}

func (s *Server) createOrAddBoardTable() error {
	tables, err := amz.ListAllTables(s.db)
	if err != nil {
		return fmt.Errorf("listing tables: %w", err)
	}

	for _, name := range tables {
		if name == tableName {
			return nil
		}
	}

	return s.createBoardTable()
}

func (s *Server) createBoardTable() error {
	if _, err := s.db.CreateTable(createTableInput()); err != nil {
		return fmt.Errorf("creating table %s: %w", tableName, err)
	}

	return nil
}

func (s *Server) addBoardSetting(b *boardSetting) error {
	if _, err := s.db.PutItem(putBoardSettingInput(b)); err != nil {
		return fmt.Errorf("adding or updating board setting: %w", err)
	}

	return nil
}

func (s *Server) deleteBoardSetting(boardID int) error {
	if _, err := s.db.DeleteItem(deleteBoardSettingInput(boardID)); err != nil {
		return fmt.Errorf("deleting board setting: %w", err)
	}

	return nil
}

func createTableInput() *dynamodb.CreateTableInput {
	return &dynamodb.CreateTableInput{
		TableName: aws.String(tableName),
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String("BoardId"),
				KeyType:       aws.String("HASH"), // Partition key
			},
		},
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String("BoardId"),
				AttributeType: aws.String("N"),
			},
		},
		BillingMode: aws.String("PAY_PER_REQUEST"),
	}
}

func putBoardSettingInput(b *boardSetting) *dynamodb.PutItemInput {
	return &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item: map[string]*dynamodb.AttributeValue{
			"BoardId": {
				N: aws.String(fmt.Sprintf("%d", b.BoardID)),
			},
			"BoardName": {
				S: aws.String(b.BoardName),
			},
			"WebexRoomId": {
				S: aws.String(b.WebexRoomID),
			},
		},
	}
}

func deleteBoardSettingInput(boardID int) *dynamodb.DeleteItemInput {
	return &dynamodb.DeleteItemInput{
		TableName: aws.String(tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"BoardId": {
				N: aws.String(strconv.Itoa(boardID)),
			},
		},
	}
}

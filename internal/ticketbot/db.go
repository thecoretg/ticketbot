package ticketbot

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"strconv"
	"tctg-automation/pkg/amz"
)

const (
	boardSettingsTableName = "ticketbot-boards"
)

// /////////////////////////////////////////////////////////////////
// Board Settings /////////////////////////////////////////////////
// ///////////////////////////////////////////////////////////////

type boardSetting struct {
	BoardID         int      `json:"board_id"`
	BoardName       string   `json:"board_name"`
	WebexRoomID     string   `json:"webex_room_id"`
	ExcludedMembers []string `json:"excluded_members"`
	Enabled         bool     `json:"enabled"`
}

func (s *Server) listBoards() ([]boardSetting, error) {
	if err := s.createOrAddBoardTable(); err != nil {
		return nil, fmt.Errorf("ensuring board table exists: %w", err)
	}

	params := &dynamodb.ScanInput{
		TableName: aws.String(boardSettingsTableName),
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

		enabled := true
		if enabledAttr := i["Enabled"]; enabledAttr != nil && enabledAttr.BOOL != nil {
			enabled = *enabledAttr.BOOL
		}

		var excludedMembers []string
		if emAttr, ok := i["ExcludedMembers"]; ok && emAttr.L != nil {
			for _, v := range emAttr.L {
				if v.S != nil {
					excludedMembers = append(excludedMembers, *v.S)
				}
			}
		}

		board := boardSetting{
			BoardID:         boardID,
			BoardName:       *i["BoardName"].S,
			WebexRoomID:     *i["WebexRoomId"].S,
			ExcludedMembers: excludedMembers,
			Enabled:         enabled,
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
		if name == boardSettingsTableName {
			return nil
		}
	}

	return s.createBoardTable()
}

func (s *Server) createBoardTable() error {
	if _, err := s.db.CreateTable(createBoardSettingsTableInput()); err != nil {
		return fmt.Errorf("creating table %s: %w", boardSettingsTableName, err)
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

func createBoardSettingsTableInput() *dynamodb.CreateTableInput {
	return &dynamodb.CreateTableInput{
		TableName: aws.String(boardSettingsTableName),
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
	excluded := make([]*dynamodb.AttributeValue, len(b.ExcludedMembers))
	for i, v := range b.ExcludedMembers {
		excluded[i] = &dynamodb.AttributeValue{S: aws.String(v)}
	}

	return &dynamodb.PutItemInput{
		TableName: aws.String(boardSettingsTableName),
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
			"ExcludedMembers": {
				L: excluded,
			},
			"Enabled:": {
				BOOL: aws.Bool(b.Enabled),
			},
		},
	}
}

func deleteBoardSettingInput(boardID int) *dynamodb.DeleteItemInput {
	return &dynamodb.DeleteItemInput{
		TableName: aws.String(boardSettingsTableName),
		Key: map[string]*dynamodb.AttributeValue{
			"BoardId": {
				N: aws.String(strconv.Itoa(boardID)),
			},
		},
	}
}

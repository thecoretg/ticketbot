package messenger

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/thecoretg/ticketbot/internal/models"
)

// sweet, man-made horrors beyond my comprehension

func createNotifierRulePayload(boards []models.Board, recips []models.WebexRecipient) json.RawMessage {
	return json.RawMessage(fmt.Sprintf(`{
	"contentType": "application/vnd.microsoft.card.adaptive",
	"content": {
		"type": "AdaptiveCard",
		"$schema": "http://adaptivecards.io/schemas/adaptive-card.json",
		"version": "1.3",
		"body": [
			{
				"type": "Input.ChoiceSet",
				"choices": [
					%s
				],
				"placeholder": "Pick a Connectwise board",
				"id": "cw_board",
				"label": "Connectwise Board",
				"isRequired": true,
				"errorMessage": "Connectwise board is required",
				"spacing": "none"
			},
			{
				"type": "Input.ChoiceSet",
				"choices": [
					%s
				],
				"placeholder": "Pick a Webex recipient",
				"id": "webex_recipient",
				"label": "Webex Recipient",
				"isRequired": true,
				"errormessage": "Webex recipient is required"
			},
			{
				"type": "TextBlock",
				"text": "By creating a notifier rule, you are enabling notifications for new tickets in your chosen ticket board to be sent to the webex recipient.",
				"wrap": true
			},
			{
				"type": "TextBlock",
				"text": "This also enables all users getting updated ticket notifications for tickets in this board that they are a resource of.",
				"wrap": true
			},
			{
				"type": "ActionSet",
				"actions": [
					{
						"type": "Action.Submit",
						"title": "Submit"
					}
				]
			}
		]
	}
}`, boardsToCardChoices(boards), recipientsToCardChoices(recips)))
}

func boardsToCardChoices(boards []models.Board) string {
	var choices []string
	for _, b := range boards {
		s := fmt.Sprintf(`{ "title": %s, "value": %s }`,
			strconv.Quote(b.Name),
			strconv.Quote(strconv.Itoa(b.ID)))
		choices = append(choices, s)
	}

	return strings.Join(choices, ",")
}

func recipientsToCardChoices(recips []models.WebexRecipient) string {
	var choices []string
	for _, b := range recips {
		nameAndType := fmt.Sprintf("%s (%s)", b.Name, b.Type)
		s := fmt.Sprintf(`{ "title": %s, "value": %s }`,
			strconv.Quote(nameAndType),
			strconv.Quote(strconv.Itoa(b.ID)))
		choices = append(choices, s)
	}

	return strings.Join(choices, ",")
}

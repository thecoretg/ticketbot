package oldserver

import (
	"fmt"
)

type ErrWasDeleted struct {
	ItemType string
	ItemID   int
}

func (e ErrWasDeleted) Error() string {
	return fmt.Sprintf("%s %d was deleted by external factors", e.ItemType, e.ItemID)
}

func errorOutput(msg string) map[string]string {
	return map[string]string{
		"error": msg,
	}
}

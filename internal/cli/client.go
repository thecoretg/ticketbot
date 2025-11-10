package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/thecoretg/ticketbot/internal/sdk"
)

func createClient(cmd *cobra.Command, args []string) error {
	var err error
	key := os.Getenv("TBOT_API_KEY")
	base := os.Getenv("TBOT_BASE_URL")

	if key == "" {
		return errors.New("api key is empty")
	}

	if base == "" {
		return errors.New("base url is empty")
	}

	client, err = sdk.NewClient(key, base)
	if err != nil {
		return fmt.Errorf("creating api client: %w", err)
	}

	return nil
}

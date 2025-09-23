package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thecoretg/ticketbot/ticketbot"
)

var (
	serverCmd = &cobra.Command{
		Use: "server",
	}

	runCmd = &cobra.Command{
		Use: "run",
		RunE: func(cmd *cobra.Command, args []string) error {
			if initAndRun {
				return ticketbot.InitAndRun(ctx)
			}

			s, err := initConfigAndServer(ctx)
			if err != nil {
				return fmt.Errorf("initializing server: %w", err)
			}

			return s.Run()
		},
	}
)

func initConfigAndServer(ctx context.Context) (*ticketbot.Server, error) {
	cfg, err := ticketbot.InitCfg()
	if err != nil {
		return nil, fmt.Errorf("initializing config: %w", err)
	}

	dbConn, err := ticketbot.ConnectToDB(ctx, cfg.PostgresDSN)
	if err != nil {
		return nil, fmt.Errorf("connecting to db: %w", err)
	}

	return ticketbot.NewServer(cfg, dbConn), nil
}

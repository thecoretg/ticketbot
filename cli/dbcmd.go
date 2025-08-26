package cli

import (
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
	"github.com/thecoretg/ticketbot/db"
	"github.com/thecoretg/ticketbot/ticketbot"
)

var (
	dbCmd = &cobra.Command{
		Use: "db",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			var err error
			cfg, err = ticketbot.InitCfg()
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			dbConn, err := pgxpool.New(ctx, cfg.PostgresDSN)
			if err != nil {
				return fmt.Errorf("connecting to database: %w", err)
			}

			queries = db.New(dbConn)
			return nil
		},
	}

	listBoardsCmd = &cobra.Command{
		Use: "list-boards",
		RunE: func(cmd *cobra.Command, args []string) error {
			boards, err := queries.ListBoards(ctx)
			if err != nil {
				return fmt.Errorf("retrieving boards from database: %w", err)
			}

			if len(boards) > 0 {
				for _, b := range boards {
					fmt.Printf("%s ID:%d, Notify Enabled:%v\n", b.Name, b.ID, b.NotifyEnabled)
				}
			} else {
				fmt.Println("No boards were found in database")
			}

			return nil
		},
	}
)

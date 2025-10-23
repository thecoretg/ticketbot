package main

import (
	"fmt"
	"sort"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/spf13/cobra"
	"github.com/thecoretg/ticketbot/internal/webex"
)

var (
	listCmd = &cobra.Command{
		Use: "list",
	}

	roomType                 string
	showRoomType, showRoomID bool
	listWebexCmd             = &cobra.Command{
		Use:     "webex-rooms",
		Aliases: []string{"wr"},
		RunE: func(cmd *cobra.Command, args []string) error {
			rooms, err := client.ListRooms(nil)
			if err != nil {
				return fmt.Errorf("listing rooms: %w", err)
			}

			rooms, err = filterWebexRooms(rooms, roomType)
			if err != nil {
				return fmt.Errorf("filtering webex rooms: %w", err)
			}

			printWebexTable(rooms)
			return nil
		},
	}
)

func addRoomsCmd() {
	rootCmd.AddCommand(listCmd)
	listCmd.AddCommand(listWebexCmd)

	listWebexCmd.Flags().StringVar(&roomType, "type", "all", "type of room to filter by: 'group', 'direct', or 'all'")
	listWebexCmd.Flags().BoolVarP(&showRoomID, "show-id", "i", true, "show room id in table")
	listWebexCmd.Flags().BoolVarP(&showRoomType, "show-type", "t", false, "show room type in table")
}

func filterWebexRooms(rooms []webex.Room, roomType string) ([]webex.Room, error) {
	if !validRoomType(roomType) {
		return nil, fmt.Errorf("room type '%s' not valid, expected 'group' or 'direct'", roomType)
	}

	if roomType == "all" {
		return rooms, nil
	}

	var filtered []webex.Room
	for _, r := range rooms {
		if r.Type == roomType {
			filtered = append(filtered, r)
		}
	}

	return filtered, nil
}

func validRoomType(t string) bool {
	return t == "group" || t == "direct" || t == "all"
}

func printWebexTable(rooms []webex.Room) {
	var (
		spacing = lipgloss.NewStyle().Padding(0, 1)
		headers = []string{"TITLE"}
	)

	if showRoomType {
		headers = append(headers, "TYPE")
	}

	if showRoomID {
		headers = append(headers, "ID")
	}

	sort.Slice(rooms, func(i, j int) bool {
		return rooms[i].Title < rooms[j].Title
	})

	t := table.New().
		Headers(headers...).
		StyleFunc(func(row, col int) lipgloss.Style {
			return spacing
		})

	for _, r := range rooms {
		// terminated users still show but with empty title, don't show them
		if r.Title != "Empty Title" {
			addRoomRow(t, r, showRoomType, showRoomID)
		}
	}

	fmt.Println(t)
}

// lowkey this sounds like something scooby doo would say
func addRoomRow(t *table.Table, room webex.Room, showType, showID bool) {
	row := []string{room.Title}
	if showType {
		row = append(row, room.Type)
	}

	if showID {
		row = append(row, room.Id)
	}

	t.Row(row...)
}

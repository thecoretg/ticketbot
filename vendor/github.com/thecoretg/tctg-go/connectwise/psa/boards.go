package psa

import (
	"context"
	"fmt"
)

func boardIDEndpoint(boardID int) string {
	return fmt.Sprintf("service/boards/%d", boardID)
}

func boardIDStatusEndpoint(boardID int) string {
	return fmt.Sprintf("%s/statuses", boardIDEndpoint(boardID))
}

func boardIDStatusIDEndpoint(boardID, statusID int) string {
	return fmt.Sprintf("%s/%d", boardIDStatusEndpoint(boardID), statusID)
}

func (c *Client) PostBoard(ctx context.Context, board *Board) (*Board, error) {
	return post[Board](ctx, c, "service/boards", board)
}

func (c *Client) ListBoards(ctx context.Context, params map[string]string) ([]Board, error) {
	return getMany[Board](ctx, c, "service/boards", params)
}

func (c *Client) GetBoard(ctx context.Context, boardID int, params map[string]string) (*Board, error) {
	return get[Board](ctx, c, boardIDEndpoint(boardID), params)
}

func (c *Client) PutBoard(ctx context.Context, boardID int, board *Board) (*Board, error) {
	return put[Board](ctx, c, boardIDEndpoint(boardID), board)
}

func (c *Client) PatchBoard(ctx context.Context, boardID int, patchOps []PatchOp) (*Board, error) {
	return patch[Board](ctx, c, boardIDEndpoint(boardID), patchOps)
}

func (c *Client) DeleteBoard(ctx context.Context, boardID int) error {
	return del(ctx, c, boardIDEndpoint(boardID))
}

func (c *Client) PostBoardStatus(ctx context.Context, boardStatus *BoardStatus, boardID int) (*BoardStatus, error) {
	return post[BoardStatus](ctx, c, boardIDStatusEndpoint(boardID), boardStatus)
}

func (c *Client) ListBoardStatuses(ctx context.Context, params map[string]string, boardID int) ([]BoardStatus, error) {
	return getMany[BoardStatus](ctx, c, boardIDStatusEndpoint(boardID), params)
}

func (c *Client) GetBoardStatus(ctx context.Context, statusID int, params map[string]string, boardID int) (*BoardStatus, error) {
	return get[BoardStatus](ctx, c, boardIDStatusIDEndpoint(boardID, statusID), params)
}

func (c *Client) PutBoardStatus(ctx context.Context, statusID int, boardStatus *BoardStatus, boardID int) (*BoardStatus, error) {
	return put[BoardStatus](ctx, c, boardIDStatusIDEndpoint(boardID, statusID), boardStatus)
}

func (c *Client) PatchBoardStatus(ctx context.Context, statusID int, patchOps []PatchOp, boardID int) (*BoardStatus, error) {
	return patch[BoardStatus](ctx, c, boardIDStatusIDEndpoint(boardID, statusID), patchOps)
}

func (c *Client) DeleteBoardStatus(ctx context.Context, statusID int, boardID int) error {
	return del(ctx, c, boardIDStatusIDEndpoint(boardID, statusID))
}

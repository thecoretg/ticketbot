package psa

import (
	"fmt"
)

func boardIdEndpoint(boardId int) string {
	return fmt.Sprintf("service/boards/%d", boardId)
}

func boardIdStatusEndpoint(boardId int) string {
	return fmt.Sprintf("%s/statuses", boardIdEndpoint(boardId))
}

func boardIdStatusIdEndpoint(boardId, statusId int) string {
	return fmt.Sprintf("%s/%d", boardIdStatusEndpoint(boardId), statusId)
}

func (c *Client) PostBoard(board *Board) (*Board, error) {
	return Post[Board](c, "service/boards", board)
}

func (c *Client) ListBoards(params map[string]string) ([]Board, error) {
	return GetMany[Board](c, "service/boards", params)
}

func (c *Client) GetBoard(boardID int, params map[string]string) (*Board, error) {
	return GetOne[Board](c, boardIdEndpoint(boardID), params)
}

func (c *Client) PutBoard(boardID int, board *Board) (*Board, error) {
	return Put[Board](c, boardIdEndpoint(boardID), board)
}

func (c *Client) PatchBoard(boardID int, patchOps []PatchOp) (*Board, error) {
	return Patch[Board](c, boardIdEndpoint(boardID), patchOps)
}

func (c *Client) DeleteBoard(boardID int) error {
	return Delete(c, boardIdEndpoint(boardID))
}

func (c *Client) PostBoardStatus(boardStatus *BoardStatus, boardID int) (*BoardStatus, error) {
	return Post[BoardStatus](c, boardIdStatusEndpoint(boardID), boardStatus)
}

func (c *Client) ListBoardStatuss(params map[string]string, boardID int) ([]BoardStatus, error) {
	return GetMany[BoardStatus](c, boardIdStatusEndpoint(boardID), params)
}

func (c *Client) GetBoardStatus(statusID int, params map[string]string, boardID int) (*BoardStatus, error) {
	return GetOne[BoardStatus](c, boardIdStatusIdEndpoint(boardID, statusID), params)
}

func (c *Client) PutBoardStatus(statusID int, boardStatus *BoardStatus, boardID int) (*BoardStatus, error) {
	return Put[BoardStatus](c, boardIdStatusIdEndpoint(boardID, statusID), boardStatus)
}

func (c *Client) PatchBoardStatus(statusID int, patchOps []PatchOp, boardID int) (*BoardStatus, error) {
	return Patch[BoardStatus](c, boardIdStatusIdEndpoint(boardID, statusID), patchOps)
}

func (c *Client) DeleteBoardStatus(statusID int, boardID int) error {
	return Delete(c, boardIdStatusIdEndpoint(boardID, statusID))
}

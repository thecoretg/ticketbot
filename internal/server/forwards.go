package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/thecoretg/ticketbot/internal/db"
)

type WebexUserForward struct {
	ID            int          `json:"id"`
	Email         string       `json:"email"`
	DestRoom      db.WebexRoom `json:"dest_room"`
	StartDate     *time.Time   `json:"start_date"`
	EndDate       *time.Time   `json:"end_date"`
	Enabled       bool         `json:"enabled"`
	UserKeepsCopy bool         `json:"user_keeps_copy"`
	CreatedOn     time.Time    `json:"created_on"`
	UpdatedOn     time.Time    `json:"updated_on"`
}

func (cl *Client) handleListUserForwards(c *gin.Context) {
	email := c.Query("email")
	fwds, err := cl.listUserForwards(c.Request.Context(), email)
	if err != nil {
		c.Error(err)
		return
	}

	if fwds == nil {
		fwds = []WebexUserForward{}
	}

	c.JSON(http.StatusOK, fwds)
}

func (cl *Client) handleGetUserForward(c *gin.Context) {
	fwdID, err := strconv.Atoi(c.Param("forward_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorOutput("forward id not provided"))
		return
	}

	f, err := cl.getUserForward(c.Request.Context(), fwdID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, errorOutput(fmt.Sprintf("forward %d not found", fwdID)))
			return
		}
		c.Error(fmt.Errorf("getting forward: %w", err))
		return
	}

	c.JSON(http.StatusOK, f)
}

func (cl *Client) handleCreateUserForward(c *gin.Context) {
	r := &db.InsertWebexUserForwardParams{}
	if err := c.ShouldBindJSON(r); err != nil {
		c.JSON(http.StatusBadRequest, errorOutput("invalid json request"))
		return
	}

	// TODO: Validate the request

	f, err := cl.createUserForward(c.Request.Context(), *r)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, f)
}

func (cl *Client) handleDeleteUserForward(c *gin.Context) {
	fwdID, err := strconv.Atoi(c.Param("forward_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorOutput("forward id not provided"))
		return
	}

	if err := cl.deleteUserForward(c.Request.Context(), fwdID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, errorOutput(fmt.Sprintf("forward %d not found", fwdID)))
			return
		}
		c.Error(fmt.Errorf("deleting forward: %w", err))
		return
	}

	c.Status(http.StatusOK)
}

func (cl *Client) listUserForwards(ctx context.Context, email string) ([]WebexUserForward, error) {
	df, err := cl.Queries.ListWebexUserForwards(ctx, strToPtr(email))
	if err != nil {
		return nil, fmt.Errorf("fetching forwards from db: %w", err)
	}

	var fwds []WebexUserForward
	for _, f := range df {
		newFwd := dbForwardToResponse(f.WebexUserForward, f.WebexRoom)
		fwds = append(fwds, *newFwd)
	}

	return fwds, nil
}

func (cl *Client) getUserForward(ctx context.Context, id int) (*WebexUserForward, error) {
	df, err := cl.Queries.GetWebexUserForward(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting forward from db: %w", err)
	}

	return dbForwardToResponse(df.WebexUserForward, df.WebexRoom), nil
}

func (cl *Client) createUserForward(ctx context.Context, p db.InsertWebexUserForwardParams) (*WebexUserForward, error) {
	f, err := cl.Queries.InsertWebexUserForward(ctx, p)
	if err != nil {
		return nil, fmt.Errorf("inserting webex user forward into store: %w", err)
	}

	df, err := cl.Queries.GetWebexUserForward(ctx, f.ID)
	if err != nil {
		return nil, fmt.Errorf("getting webex user forward (create was successful): %w", err)
	}

	return dbForwardToResponse(df.WebexUserForward, df.WebexRoom), nil
}

func (cl *Client) deleteUserForward(ctx context.Context, id int) error {
	return cl.Queries.DeleteWebexForward(ctx, id)
}

func dbForwardToResponse(uf db.WebexUserForward, r db.WebexRoom) *WebexUserForward {
	return &WebexUserForward{
		ID:            uf.ID,
		Email:         uf.UserEmail,
		DestRoom:      r,
		StartDate:     uf.StartDate,
		EndDate:       uf.EndDate,
		Enabled:       uf.Enabled,
		UserKeepsCopy: uf.UserKeepsCopy,
		CreatedOn:     uf.CreatedOn,
		UpdatedOn:     uf.UpdatedOn,
	}
}

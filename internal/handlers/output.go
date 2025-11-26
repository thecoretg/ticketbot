package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Output wrappers for consistent json output in handlers

type APIError struct {
	Message string `json:"error"`
}

func (e *APIError) Error() string {
	return e.Message
}

type ResultOutput struct {
	Result string `json:"result"`
}

func outputJSON(c *gin.Context, o any) {
	c.JSON(http.StatusOK, o)
}

func resultJSON(c *gin.Context, result string) {
	c.JSON(http.StatusOK, ResultOutput{Result: result})
}

func conflictError(c *gin.Context, err error) {
	errJSON(c, http.StatusConflict, err)
}

func internalServerError(c *gin.Context, err error) {
	errJSON(c, http.StatusInternalServerError, err)
}

func badPayloadError(c *gin.Context, err error) {
	e := fmt.Errorf("bad json request: %w", err)
	errJSON(c, http.StatusBadRequest, e)
}

func notFoundError(c *gin.Context, err error) {
	errJSON(c, http.StatusNotFound, err)
}

func badIntError(c *gin.Context) {
	s := c.Param("id")
	errJSON(c, http.StatusBadRequest, fmt.Errorf("%s is not a valid integer", s))
}

func errJSON(c *gin.Context, code int, err error) {
	c.JSON(code, APIError{Message: err.Error()})
}

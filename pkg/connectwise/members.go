package connectwise

import (
	"context"
	"fmt"
)

func memberIdEndpoint(memberId int) string {
	return fmt.Sprintf("system/members/%d", memberId)
}

func (c *Client) ListMembers(ctx context.Context, params *QueryParams) ([]Member, error) {
	return ApiRequestPaginated[Member](ctx, c, "GET", "system/members", params, nil)
}

func (c *Client) GetMember(ctx context.Context, memberId int, params *QueryParams) (*Member, error) {
	return ApiRequestNonPaginated[Member](ctx, c, "GET", memberIdEndpoint(memberId), params, nil)
}

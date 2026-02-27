package psa

import (
	"fmt"
)

func memberIdEndpoint(memberId int) string {
	return fmt.Sprintf("system/members/%d", memberId)
}

func (c *Client) PostMember(member *Member) (*Member, error) {
	return Post[Member](c, "system/members", member)
}

func (c *Client) ListMembers(params map[string]string) ([]Member, error) {
	return GetMany[Member](c, "system/members", params)
}

func (c *Client) GetMemberByIdentifier(identifier string) (*Member, error) {
	p := map[string]string{
		"conditions": fmt.Sprintf("identifier='%s'", identifier),
	}

	members, err := c.ListMembers(p)
	if err != nil {
		return nil, fmt.Errorf("listing members: %w", err)
	}

	if len(members) == 0 {
		return nil, ErrNotFound
	}

	if len(members) > 1 {
		return nil, fmt.Errorf("expected 1 member for identifier %s, received %d", identifier, len(members))
	}

	return &members[0], nil
}

func (c *Client) GetMember(memberID int, params map[string]string) (*Member, error) {
	return GetOne[Member](c, memberIdEndpoint(memberID), params)
}

func (c *Client) PutMember(memberID int, member *Member) (*Member, error) {
	return Put[Member](c, memberIdEndpoint(memberID), member)
}

func (c *Client) PatchMember(memberID int, patchOps []PatchOp) (*Member, error) {
	return Patch[Member](c, memberIdEndpoint(memberID), patchOps)
}

func (c *Client) DeleteMember(memberID int) error {
	return Delete(c, memberIdEndpoint(memberID))
}

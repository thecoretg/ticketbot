package psa

import (
	"context"
	"fmt"
)

func contactIDEndpoint(contactID int) string {
	return fmt.Sprintf("company/contacts/%d", contactID)
}

func (c *Client) PostContact(ctx context.Context, contact *Contact) (*Contact, error) {
	return post[Contact](ctx, c, "company/contacts", contact)
}

func (c *Client) ListContacts(ctx context.Context, params map[string]string) ([]Contact, error) {
	return getMany[Contact](ctx, c, "company/contacts", params)
}

func (c *Client) GetContact(ctx context.Context, contactID int, params map[string]string) (*Contact, error) {
	return get[Contact](ctx, c, contactIDEndpoint(contactID), params)
}

func (c *Client) PutContact(ctx context.Context, contactID int, contact *Contact) (*Contact, error) {
	return put[Contact](ctx, c, contactIDEndpoint(contactID), contact)
}

func (c *Client) PatchContact(ctx context.Context, contactID int, patchOps []PatchOp) (*Contact, error) {
	return patch[Contact](ctx, c, contactIDEndpoint(contactID), patchOps)
}

func (c *Client) DeleteContact(ctx context.Context, contactID int) error {
	return del(ctx, c, contactIDEndpoint(contactID))
}

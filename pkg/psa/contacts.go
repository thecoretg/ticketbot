package psa

import (
	"fmt"
)

func contactIdEndpoint(contactId int) string {
	return fmt.Sprintf("company/contacts/%d", contactId)
}

func (c *Client) PostContact(contact *Contact) (*Contact, error) {
	return Post[Contact](c, "company/contacts", contact)
}

func (c *Client) ListContacts(params map[string]string) ([]Contact, error) {
	return GetMany[Contact](c, "company/contacts", params)
}

func (c *Client) GetContact(contactID int, params map[string]string) (*Contact, error) {
	return GetOne[Contact](c, contactIdEndpoint(contactID), params)
}

func (c *Client) PutContact(contactID int, contact *Contact) (*Contact, error) {
	return Put[Contact](c, contactIdEndpoint(contactID), contact)
}

func (c *Client) PatchContact(contactID int, patchOps []PatchOp) (*Contact, error) {
	return Patch[Contact](c, contactIdEndpoint(contactID), patchOps)
}

func (c *Client) DeleteContact(contactID int) error {
	return Delete(c, contactIdEndpoint(contactID))
}

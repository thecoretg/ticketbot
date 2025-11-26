package psa

import (
	"fmt"
)

func companyIdEndpoint(companyId int) string {
	return fmt.Sprintf("company/companies/%d", companyId)
}

func (c *Client) PostCompany(company *Company) (*Company, error) {
	return Post[Company](c, "company/companies", company)
}

func (c *Client) ListCompanies(params map[string]string) ([]Company, error) {
	return GetMany[Company](c, "company/companies", params)
}

func (c *Client) GetCompany(companyID int, params map[string]string) (*Company, error) {
	return GetOne[Company](c, companyIdEndpoint(companyID), params)
}

func (c *Client) PutCompany(companyID int, company *Company) (*Company, error) {
	return Put[Company](c, companyIdEndpoint(companyID), company)
}

func (c *Client) PatchCompany(companyID int, patchOps []PatchOp) (*Company, error) {
	return Patch[Company](c, companyIdEndpoint(companyID), patchOps)
}

func (c *Client) DeleteCompany(companyID int) error {
	return Delete(c, companyIdEndpoint(companyID))
}

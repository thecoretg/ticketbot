package workflow

// ConditionField documents a commonly used condition path for editor autocomplete.
type ConditionField struct {
	Path        string `json:"path"`
	Type        string `json:"type"` // string | number | bool
	Description string `json:"description"`
	Example     string `json:"example"`
}

// ConditionFields lists the ticket paths most rules are written against. Any field on the
// ConnectWise ticket JSON works; this is guidance, not a whitelist.
var ConditionFields = []ConditionField{
	{"id", "number", "Ticket number", "id = 12345"},
	{"summary", "string", "Ticket summary", "summary contains 'vpn'"},
	{"board/id", "number", "Board id", "board/id = 34"},
	{"board/name", "string", "Board name", "board/name = 'Help Desk'"},
	{"status/id", "number", "Status id", "status/id = 16"},
	{"status/name", "string", "Status name", "status/name in ('New', 'Assigned')"},
	{"priority/id", "number", "Priority id", "priority/id = 4"},
	{"priority/name", "string", "Priority name", "priority/name like 'Priority 1*'"},
	{"company/id", "number", "Company id", "company/id = 250"},
	{"company/identifier", "string", "Company identifier", "company/identifier = 'ACME'"},
	{"company/name", "string", "Company name", "company/name contains 'Acme'"},
	{"contact/id", "number", "Contact id", "contact/id = 900"},
	{"contact/name", "string", "Contact name", "contact/name contains 'Smith'"},
	{"owner/id", "number", "Owner member id", "owner/id != null"},
	{"owner/identifier", "string", "Owner member identifier", "owner/identifier = 'jdoe'"},
	{"resources", "string", "Comma-separated resource identifiers", "resources contains 'jdoe'"},
	{"type/name", "string", "Ticket type", "type/name = 'Incident'"},
	{"subType/name", "string", "Ticket subtype", "subType/name = 'Network'"},
	{"item/name", "string", "Ticket item", "item/name = 'VPN'"},
	{"closedFlag", "bool", "Ticket is closed", "closedFlag = true"},
	{"_info/updatedBy", "string", "Identifier of the last member to update the ticket", "_info/updatedBy != 'ticketbot'"},
	{"latestNote/text", "string", "Text of the note that triggered this run", "latestNote/text contains 'urgent'"},
	{"latestNote/internalAnalysisFlag", "bool", "Triggering note is internal", "latestNote/internalAnalysisFlag = true"},
	{"latestNote/member/identifier", "string", "Member who wrote the triggering note", "latestNote/member/identifier = 'jdoe'"},
	{"latestNote/contact/id", "number", "Contact who wrote the triggering note (customer replies)", "latestNote/contact/id != null"},
	{"changed/status", "bool", "Status changed in this update", "changed/status = true"},
	{"changed/priority", "bool", "Priority changed in this update", "changed/priority = true"},
	{"changed/owner", "bool", "Owner changed in this update", "changed/owner = true"},
	{"changed/resources", "bool", "Resources changed in this update", "changed/resources = true"},
	{"changed/board", "bool", "Ticket moved boards in this update", "changed/board = true"},
	{"changed/summary", "bool", "Summary changed in this update", "changed/summary = true"},
	{"changed/closedFlag", "bool", "Ticket opened or closed in this update", "changed/closedFlag = true"},
}

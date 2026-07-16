package psa

import "time"

type Board struct {
	Info                           any    `json:"_info,omitempty"`
	AllSort                        string `json:"allSort,omitempty"`
	AutoAssignLimitAmount          int    `json:"autoAssignLimitAmount,omitempty"`
	AutoAssignLimitFlag            bool   `json:"autoAssignLimitFlag,omitempty"`
	AutoAssignNewECTicketsFlag     bool   `json:"autoAssignNewECTicketsFlag,omitempty"`
	AutoAssignNewPortalTicketsFlag bool   `json:"autoAssignNewPortalTicketsFlag,omitempty"`
	AutoAssignNewTicketsFlag       bool   `json:"autoAssignNewTicketsFlag,omitempty"`
	AutoAssignTicketOwnerFlag      bool   `json:"autoAssignTicketOwnerFlag,omitempty"`
	AutoCloseStatus                struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
		Sort int    `json:"sort,omitempty"`
	} `json:"autoCloseStatus,omitzero"`
	BillExpense                   string `json:"billExpense,omitempty"`
	BillProduct                   string `json:"billProduct,omitempty"`
	BillTicketSeparatelyFlag      bool   `json:"billTicketSeparatelyFlag,omitempty"`
	BillTicketsAfterClosedFlag    bool   `json:"billTicketsAfterClosedFlag,omitempty"`
	BillTime                      string `json:"billTime,omitempty"`
	BillUnapprovedTimeExpenseFlag bool   `json:"billUnapprovedTimeExpenseFlag,omitempty"`
	BoardIcon                     struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"boardIcon,omitzero"`
	ClosedLoopAllFlag              bool `json:"closedLoopAllFlag,omitempty"`
	ClosedLoopDiscussionsFlag      bool `json:"closedLoopDiscussionsFlag,omitempty"`
	ClosedLoopInternalAnalysisFlag bool `json:"closedLoopInternalAnalysisFlag,omitempty"`
	ClosedLoopResolutionFlag       bool `json:"closedLoopResolutionFlag,omitempty"`
	ContactTemplate                struct {
		Info       any    `json:"_info,omitempty"`
		ID         int    `json:"id,omitempty"`
		Identifier string `json:"identifier,omitempty"`
		Name       string `json:"name,omitempty"`
		Type       string `json:"type,omitempty"`
	} `json:"contactTemplate,omitzero"`
	Department struct {
		Info       any    `json:"_info,omitempty"`
		ID         int    `json:"id,omitempty"`
		Identifier string `json:"identifier,omitempty"`
		Name       string `json:"name,omitempty"`
	} `json:"department"`
	DiscussionsLockedFlag bool `json:"discussionsLockedFlag,omitempty"`
	DispatchMember        struct {
		Info          any     `json:"_info,omitempty"`
		DailyCapacity float64 `json:"dailyCapacity,omitempty"`
		ID            int     `json:"id,omitempty"`
		Identifier    string  `json:"identifier,omitempty"`
		Name          string  `json:"name,omitempty"`
	} `json:"dispatchMember,omitzero"`
	DutyManagerMember struct {
		Info          any     `json:"_info,omitempty"`
		DailyCapacity float64 `json:"dailyCapacity,omitempty"`
		ID            int     `json:"id,omitempty"`
		Identifier    string  `json:"identifier,omitempty"`
		Name          string  `json:"name,omitempty"`
	} `json:"dutyManagerMember,omitzero"`
	EmailConnectorAllowReopenClosedFlag       bool `json:"emailConnectorAllowReopenClosedFlag,omitempty"`
	EmailConnectorNeverReopenByDaysClosedFlag bool `json:"emailConnectorNeverReopenByDaysClosedFlag,omitempty"`
	EmailConnectorNeverReopenByDaysFlag       bool `json:"emailConnectorNeverReopenByDaysFlag,omitempty"`
	EmailConnectorNewTicketNoMatchFlag        bool `json:"emailConnectorNewTicketNoMatchFlag,omitempty"`
	EmailConnectorReopenDaysClosedLimit       int  `json:"emailConnectorReopenDaysClosedLimit,omitempty"`
	EmailConnectorReopenDaysLimit             int  `json:"emailConnectorReopenDaysLimit,omitempty"`
	EmailConnectorReopenResourcesFlag         bool `json:"emailConnectorReopenResourcesFlag,omitempty"`
	EmailConnectorReopenStatus                struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
		Sort int    `json:"sort,omitempty"`
	} `json:"emailConnectorReopenStatus,omitzero"`
	ID                   int    `json:"id,omitempty"`
	InactiveFlag         bool   `json:"inactiveFlag,omitempty"`
	InternalAnalysisSort string `json:"internalAnalysisSort,omitempty"`
	Location             struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"location"`
	MarkFirstNoteIssueFlag bool   `json:"markFirstNoteIssueFlag,omitempty"`
	Name                   string `json:"name"`
	NotifyEmailFrom        string `json:"notifyEmailFrom,omitempty"`
	NotifyEmailFromName    string `json:"notifyEmailFromName,omitempty"`
	OncallMember           struct {
		Info          any     `json:"_info,omitempty"`
		DailyCapacity float64 `json:"dailyCapacity,omitempty"`
		ID            int     `json:"id,omitempty"`
		Identifier    string  `json:"identifier,omitempty"`
		Name          string  `json:"name,omitempty"`
	} `json:"oncallMember,omitzero"`
	OverrideBillingSetupFlag bool   `json:"overrideBillingSetupFlag,omitempty"`
	PercentageCalculation    string `json:"percentageCalculation,omitempty"`
	ProblemSort              string `json:"problemSort,omitempty"`
	ProjectFlag              bool   `json:"projectFlag,omitempty"`
	ResolutionSort           string `json:"resolutionSort,omitempty"`
	ResourceTemplate         struct {
		Info       any    `json:"_info,omitempty"`
		ID         int    `json:"id,omitempty"`
		Identifier string `json:"identifier,omitempty"`
		Name       string `json:"name,omitempty"`
		Type       string `json:"type,omitempty"`
	} `json:"resourceTemplate,omitzero"`
	RestrictBoardByDefaultFlag bool `json:"restrictBoardByDefaultFlag,omitempty"`
	SendToBundledFlag          bool `json:"sendToBundledFlag,omitempty"`
	SendToCCFlag               bool `json:"sendToCCFlag,omitempty"`
	SendToContactFlag          bool `json:"sendToContactFlag,omitempty"`
	SendToResourceFlag         bool `json:"sendToResourceFlag,omitempty"`
	ServiceManagerMember       struct {
		Info          any     `json:"_info,omitempty"`
		DailyCapacity float64 `json:"dailyCapacity,omitempty"`
		ID            int     `json:"id,omitempty"`
		Identifier    string  `json:"identifier,omitempty"`
		Name          string  `json:"name,omitempty"`
	} `json:"serviceManagerMember,omitzero"`
	ShowDependenciesFlag bool `json:"showDependenciesFlag,omitempty"`
	ShowEstimatesFlag    bool `json:"showEstimatesFlag,omitempty"`
	SignOffTemplate      struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"signOffTemplate,omitzero"`
	TimeEntryDiscussionFlag       bool `json:"timeEntryDiscussionFlag,omitempty"`
	TimeEntryInternalAnalysisFlag bool `json:"timeEntryInternalAnalysisFlag,omitempty"`
	TimeEntryLockedFlag           bool `json:"timeEntryLockedFlag,omitempty"`
	TimeEntryResolutionFlag       bool `json:"timeEntryResolutionFlag,omitempty"`
	UseMemberDisplayNameFlag      bool `json:"useMemberDisplayNameFlag,omitempty"`
	WorkRole                      struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"workRole,omitzero"`
	WorkType struct {
		Info            any    `json:"_info,omitempty"`
		ID              int    `json:"id,omitempty"`
		Name            string `json:"name,omitempty"`
		UtilizationFlag bool   `json:"utilizationFlag,omitempty"`
	} `json:"workType,omitzero"`
}

type BoardStatus struct {
	Info  any `json:"_info,omitempty"`
	Board struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"board,omitzero"`
	ClosedStatus              bool   `json:"closedStatus,omitempty"`
	CustomStatusIndicatorName string `json:"customStatusIndicatorName,omitempty"`
	CustomerPortalDescription string `json:"customerPortalDescription,omitempty"`
	CustomerPortalFlag        bool   `json:"customerPortalFlag,omitempty"`
	DefaultFlag               bool   `json:"defaultFlag,omitempty"`
	DisplayOnBoard            bool   `json:"displayOnBoard,omitempty"`
	EmailTemplate             struct {
		Info       any    `json:"_info,omitempty"`
		ID         int    `json:"id,omitempty"`
		Identifier string `json:"identifier,omitempty"`
		Name       string `json:"name,omitempty"`
		Type       string `json:"type,omitempty"`
	} `json:"emailTemplate,omitzero"`
	EscalationStatus   string `json:"escalationStatus,omitempty"`
	ID                 int    `json:"id,omitempty"`
	Inactive           bool   `json:"inactive,omitempty"`
	Name               string `json:"name"`
	RoundRobinCatchall bool   `json:"roundRobinCatchall,omitempty"`
	SaveTimeAsNote     bool   `json:"saveTimeAsNote,omitempty"`
	SortOrder          int    `json:"sortOrder,omitempty"`
	StatusIndicator    struct {
		Info       any    `json:"_info,omitempty"`
		ID         int    `json:"id,omitempty"`
		Identifier string `json:"identifier,omitempty"`
		Name       string `json:"name,omitempty"`
	} `json:"statusIndicator,omitzero"`
	TimeEntryNotAllowed bool `json:"timeEntryNotAllowed,omitempty"`
}

type Callback struct {
	Info                 any    `json:"_info,omitempty"`
	ConnectWiseID        string `json:"connectWiseID,omitempty"`
	Description          string `json:"description,omitempty"`
	ID                   int    `json:"id,omitempty"`
	InactiveFlag         bool   `json:"inactiveFlag,omitempty"`
	IsSelfSuppressedFlag bool   `json:"isSelfSuppressedFlag,omitempty"`
	IsSoapCallbackFlag   bool   `json:"isSoapCallbackFlag,omitempty"`
	Level                string `json:"level"`
	MemberID             int    `json:"memberId,omitempty"`
	ObjectID             int    `json:"objectId,omitempty"`
	PayloadVersion       string `json:"payloadVersion,omitempty"`
	Type                 string `json:"type"`
	URL                  string `json:"url"`
}

type Company struct {
	ID         int    `json:"id,omitempty"`
	Identifier string `json:"identifier"`
	Name       string `json:"name"`
	Status     struct {
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
		Info struct {
			AdditionalProp1 string `json:"additionalProp1,omitempty"`
			AdditionalProp2 string `json:"additionalProp2,omitempty"`
			AdditionalProp3 string `json:"additionalProp3,omitempty"`
		} `json:"_info,omitzero"`
	} `json:"status,omitzero"`
	AddressLine1 string `json:"addressLine1,omitempty"`
	AddressLine2 string `json:"addressLine2,omitempty"`
	City         string `json:"city,omitempty"`
	State        string `json:"state,omitempty"`
	Zip          string `json:"zip,omitempty"`
	Country      struct {
		ID         int    `json:"id,omitempty"`
		Identifier string `json:"identifier,omitempty"`
		Name       string `json:"name,omitempty"`
		Info       struct {
			AdditionalProp1 string `json:"additionalProp1,omitempty"`
			AdditionalProp2 string `json:"additionalProp2,omitempty"`
			AdditionalProp3 string `json:"additionalProp3,omitempty"`
		} `json:"_info,omitzero"`
	} `json:"country,omitzero"`
	PhoneNumber string `json:"phoneNumber,omitempty"`
	FaxNumber   string `json:"faxNumber,omitempty"`
	Website     string `json:"website,omitempty"`
	Territory   struct {
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
		Info struct {
			AdditionalProp1 string `json:"additionalProp1,omitempty"`
			AdditionalProp2 string `json:"additionalProp2,omitempty"`
			AdditionalProp3 string `json:"additionalProp3,omitempty"`
		} `json:"_info,omitzero"`
	} `json:"territory,omitzero"`
	Market struct {
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
		Info struct {
			AdditionalProp1 string `json:"additionalProp1,omitempty"`
			AdditionalProp2 string `json:"additionalProp2,omitempty"`
			AdditionalProp3 string `json:"additionalProp3,omitempty"`
		} `json:"_info,omitzero"`
	} `json:"market,omitzero"`
	AccountNumber  string `json:"accountNumber,omitempty"`
	DefaultContact struct {
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
		Info struct {
			AdditionalProp1 string `json:"additionalProp1,omitempty"`
			AdditionalProp2 string `json:"additionalProp2,omitempty"`
			AdditionalProp3 string `json:"additionalProp3,omitempty"`
		} `json:"_info,omitzero"`
	} `json:"defaultContact,omitzero"`
	DateAcquired time.Time `json:"dateAcquired,omitzero"`
	SicCode      struct {
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
		Info struct {
			AdditionalProp1 string `json:"additionalProp1,omitempty"`
			AdditionalProp2 string `json:"additionalProp2,omitempty"`
			AdditionalProp3 string `json:"additionalProp3,omitempty"`
		} `json:"_info,omitzero"`
	} `json:"sicCode,omitzero"`
	ParentCompany struct {
		ID         int    `json:"id,omitempty"`
		Identifier string `json:"identifier,omitempty"`
		Name       string `json:"name,omitempty"`
		Info       struct {
			AdditionalProp1 string `json:"additionalProp1,omitempty"`
			AdditionalProp2 string `json:"additionalProp2,omitempty"`
			AdditionalProp3 string `json:"additionalProp3,omitempty"`
		} `json:"_info,omitzero"`
	} `json:"parentCompany,omitzero"`
	AnnualRevenue     float64 `json:"annualRevenue,omitempty"`
	NumberOfEmployees int     `json:"numberOfEmployees,omitempty"`
	YearEstablished   int     `json:"yearEstablished,omitempty"`
	RevenueYear       int     `json:"revenueYear,omitempty"`
	OwnershipType     struct {
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
		Info struct {
			AdditionalProp1 string `json:"additionalProp1,omitempty"`
			AdditionalProp2 string `json:"additionalProp2,omitempty"`
			AdditionalProp3 string `json:"additionalProp3,omitempty"`
		} `json:"_info,omitzero"`
	} `json:"ownershipType,omitzero"`
	TimeZoneSetup struct {
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
		Info struct {
			AdditionalProp1 string `json:"additionalProp1,omitempty"`
			AdditionalProp2 string `json:"additionalProp2,omitempty"`
			AdditionalProp3 string `json:"additionalProp3,omitempty"`
		} `json:"_info,omitzero"`
	} `json:"timeZoneSetup,omitzero"`
	LeadSource      string `json:"leadSource,omitempty"`
	LeadFlag        bool   `json:"leadFlag,omitempty"`
	UnsubscribeFlag bool   `json:"unsubscribeFlag,omitempty"`
	Calendar        struct {
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
		Info struct {
			AdditionalProp1 string `json:"additionalProp1,omitempty"`
			AdditionalProp2 string `json:"additionalProp2,omitempty"`
			AdditionalProp3 string `json:"additionalProp3,omitempty"`
		} `json:"_info,omitzero"`
	} `json:"calendar,omitzero"`
	UserDefinedField1  string `json:"userDefinedField1,omitempty"`
	UserDefinedField2  string `json:"userDefinedField2,omitempty"`
	UserDefinedField3  string `json:"userDefinedField3,omitempty"`
	UserDefinedField4  string `json:"userDefinedField4,omitempty"`
	UserDefinedField5  string `json:"userDefinedField5,omitempty"`
	UserDefinedField6  string `json:"userDefinedField6,omitempty"`
	UserDefinedField7  string `json:"userDefinedField7,omitempty"`
	UserDefinedField8  string `json:"userDefinedField8,omitempty"`
	UserDefinedField9  string `json:"userDefinedField9,omitempty"`
	UserDefinedField10 string `json:"userDefinedField10,omitempty"`
	VendorIdentifier   string `json:"vendorIdentifier,omitempty"`
	TaxIdentifier      string `json:"taxIdentifier,omitempty"`
	TaxCode            struct {
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
		Info struct {
			AdditionalProp1 string `json:"additionalProp1,omitempty"`
			AdditionalProp2 string `json:"additionalProp2,omitempty"`
			AdditionalProp3 string `json:"additionalProp3,omitempty"`
		} `json:"_info,omitzero"`
	} `json:"taxCode,omitzero"`
	BillingTerms struct {
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
		Info struct {
			AdditionalProp1 string `json:"additionalProp1,omitempty"`
			AdditionalProp2 string `json:"additionalProp2,omitempty"`
			AdditionalProp3 string `json:"additionalProp3,omitempty"`
		} `json:"_info,omitzero"`
	} `json:"billingTerms,omitzero"`
	InvoiceTemplate struct {
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
		Info struct {
			AdditionalProp1 string `json:"additionalProp1,omitempty"`
			AdditionalProp2 string `json:"additionalProp2,omitempty"`
			AdditionalProp3 string `json:"additionalProp3,omitempty"`
		} `json:"_info,omitzero"`
	} `json:"invoiceTemplate,omitzero"`
	PricingSchedule struct {
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
		Info struct {
			AdditionalProp1 string `json:"additionalProp1,omitempty"`
			AdditionalProp2 string `json:"additionalProp2,omitempty"`
			AdditionalProp3 string `json:"additionalProp3,omitempty"`
		} `json:"_info,omitzero"`
	} `json:"pricingSchedule,omitzero"`
	CompanyEntityType struct {
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
		Info struct {
			AdditionalProp1 string `json:"additionalProp1,omitempty"`
			AdditionalProp2 string `json:"additionalProp2,omitempty"`
			AdditionalProp3 string `json:"additionalProp3,omitempty"`
		} `json:"_info,omitzero"`
	} `json:"companyEntityType,omitzero"`
	BillToCompany struct {
		ID         int    `json:"id,omitempty"`
		Identifier string `json:"identifier,omitempty"`
		Name       string `json:"name,omitempty"`
		Info       struct {
			AdditionalProp1 string `json:"additionalProp1,omitempty"`
			AdditionalProp2 string `json:"additionalProp2,omitempty"`
			AdditionalProp3 string `json:"additionalProp3,omitempty"`
		} `json:"_info,omitzero"`
	} `json:"billToCompany,omitzero"`
	BillingSite struct {
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
		Info struct {
			AdditionalProp1 string `json:"additionalProp1,omitempty"`
			AdditionalProp2 string `json:"additionalProp2,omitempty"`
			AdditionalProp3 string `json:"additionalProp3,omitempty"`
		} `json:"_info,omitzero"`
	} `json:"billingSite,omitzero"`
	BillingContact struct {
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
		Info struct {
			AdditionalProp1 string `json:"additionalProp1,omitempty"`
			AdditionalProp2 string `json:"additionalProp2,omitempty"`
			AdditionalProp3 string `json:"additionalProp3,omitempty"`
		} `json:"_info,omitzero"`
	} `json:"billingContact,omitzero"`
	InvoiceDeliveryMethod struct {
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
		Info struct {
			AdditionalProp1 string `json:"additionalProp1,omitempty"`
			AdditionalProp2 string `json:"additionalProp2,omitempty"`
			AdditionalProp3 string `json:"additionalProp3,omitempty"`
		} `json:"_info,omitzero"`
	} `json:"invoiceDeliveryMethod,omitzero"`
	InvoiceToEmailAddress string    `json:"invoiceToEmailAddress,omitempty"`
	InvoiceCCEmailAddress string    `json:"invoiceCCEmailAddress,omitempty"`
	DeletedFlag           bool      `json:"deletedFlag,omitempty"`
	DateDeleted           time.Time `json:"dateDeleted,omitzero"`
	DeletedBy             string    `json:"deletedBy,omitempty"`
	MobileGUID            string    `json:"mobileGuid,omitempty"`
	FacebookURL           string    `json:"facebookUrl,omitempty"`
	TwitterURL            string    `json:"twitterUrl,omitempty"`
	LinkedInURL           string    `json:"linkedInUrl,omitempty"`
	Currency              struct {
		ID                      int    `json:"id,omitempty"`
		Symbol                  string `json:"symbol,omitempty"`
		CurrencyCode            string `json:"currencyCode,omitempty"`
		DecimalSeparator        string `json:"decimalSeparator,omitempty"`
		NumberOfDecimals        int    `json:"numberOfDecimals,omitempty"`
		ThousandsSeparator      string `json:"thousandsSeparator,omitempty"`
		NegativeParenthesesFlag bool   `json:"negativeParenthesesFlag,omitempty"`
		DisplaySymbolFlag       bool   `json:"displaySymbolFlag,omitempty"`
		CurrencyIdentifier      string `json:"currencyIdentifier,omitempty"`
		DisplayIDFlag           bool   `json:"displayIdFlag,omitempty"`
		RightAlign              bool   `json:"rightAlign,omitempty"`
		Name                    string `json:"name,omitempty"`
		Info                    struct {
			AdditionalProp1 string `json:"additionalProp1,omitempty"`
			AdditionalProp2 string `json:"additionalProp2,omitempty"`
			AdditionalProp3 string `json:"additionalProp3,omitempty"`
		} `json:"_info,omitzero"`
	} `json:"currency,omitzero"`
	TerritoryManager struct {
		ID            int    `json:"id,omitempty"`
		Identifier    string `json:"identifier,omitempty"`
		Name          string `json:"name,omitempty"`
		DailyCapacity int    `json:"dailyCapacity,omitempty"`
		Info          struct {
			AdditionalProp1 string `json:"additionalProp1,omitempty"`
			AdditionalProp2 string `json:"additionalProp2,omitempty"`
			AdditionalProp3 string `json:"additionalProp3,omitempty"`
		} `json:"_info,omitzero"`
	} `json:"territoryManager,omitzero"`
	ResellerIdentifier string `json:"resellerIdentifier,omitempty"`
	IsVendorFlag       bool   `json:"isVendorFlag,omitempty"`
	Types              []struct {
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
		Info struct {
			AdditionalProp1 string `json:"additionalProp1,omitempty"`
			AdditionalProp2 string `json:"additionalProp2,omitempty"`
			AdditionalProp3 string `json:"additionalProp3,omitempty"`
		} `json:"_info,omitzero"`
	} `json:"types,omitzero"`
	Site struct {
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
		Info struct {
			AdditionalProp1 string `json:"additionalProp1,omitempty"`
			AdditionalProp2 string `json:"additionalProp2,omitempty"`
			AdditionalProp3 string `json:"additionalProp3,omitempty"`
		} `json:"_info,omitzero"`
	} `json:"site,omitzero"`
	IntegratorTags []string `json:"integratorTags,omitempty"`
	Info           struct {
		AdditionalProp1 string `json:"additionalProp1,omitempty"`
		AdditionalProp2 string `json:"additionalProp2,omitempty"`
		AdditionalProp3 string `json:"additionalProp3,omitempty"`
	} `json:"_info,omitzero"`
	CustomFields []struct {
		ID               int    `json:"id,omitempty"`
		Caption          string `json:"caption,omitempty"`
		Type             string `json:"type,omitempty"`
		EntryMethod      string `json:"entryMethod,omitempty"`
		NumberOfDecimals int    `json:"numberOfDecimals,omitempty"`
		Value            any    `json:"value,omitempty"`
		ConnectWiseID    string `json:"connectWiseId,omitempty"`
	} `json:"customFields,omitzero"`
}
type Contact struct {
	Info             any    `json:"_info,omitempty"`
	AddressLine1     string `json:"addressLine1,omitempty"`
	AddressLine2     string `json:"addressLine2,omitempty"`
	Anniversary      string `json:"anniversary,omitempty"`
	AssistantContact struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"assistantContact,omitzero"`
	BirthDay           string `json:"birthDay,omitempty"`
	Children           string `json:"children,omitempty"`
	ChildrenFlag       bool   `json:"childrenFlag,omitempty"`
	City               string `json:"city,omitempty"`
	CommunicationItems []struct {
		CommunicationType string `json:"communicationType,omitempty"`
		DefaultFlag       bool   `json:"defaultFlag,omitempty"`
		Domain            string `json:"domain,omitempty"`
		Extension         string `json:"extension,omitempty"`
		ID                int    `json:"id,omitempty"`
		Type              struct {
			Info any    `json:"_info,omitempty"`
			ID   int    `json:"id,omitempty"`
			Name string `json:"name,omitempty"`
		} `json:"type,omitzero"`
		Value string `json:"value,omitempty"`
	} `json:"communicationItems,omitzero"`
	Company struct {
		Info       any    `json:"_info,omitempty"`
		ID         int    `json:"id,omitempty"`
		Identifier string `json:"identifier,omitempty"`
		Name       string `json:"name,omitempty"`
	} `json:"company,omitzero"`
	CompanyLocation struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"companyLocation,omitzero"`
	Country struct {
		Info       any    `json:"_info,omitempty"`
		ID         int    `json:"id,omitempty"`
		Identifier string `json:"identifier,omitempty"`
		Name       string `json:"name,omitempty"`
	} `json:"country,omitzero"`
	CustomFields []struct {
		Caption          string `json:"caption,omitempty"`
		ConnectWiseID    string `json:"connectWiseId,omitempty"`
		EntryMethod      string `json:"entryMethod,omitempty"`
		ID               int    `json:"id,omitempty"`
		NumberOfDecimals int    `json:"numberOfDecimals,omitempty"`
		Type             string `json:"type,omitempty"`
		Value            any    `json:"value,omitempty"`
	} `json:"customFields,omitzero"`
	DefaultBillingFlag    bool   `json:"defaultBillingFlag,omitempty"`
	DefaultFlag           bool   `json:"defaultFlag,omitempty"`
	DefaultMergeContactID int    `json:"defaultMergeContactId,omitempty"`
	DefaultPhoneExtension string `json:"defaultPhoneExtension,omitempty"`
	DefaultPhoneNbr       string `json:"defaultPhoneNbr,omitempty"`
	DefaultPhoneType      string `json:"defaultPhoneType,omitempty"`
	Department            struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"department,omitzero"`
	DisablePortalLoginFlag bool     `json:"disablePortalLoginFlag,omitempty"`
	FacebookURL            string   `json:"facebookUrl,omitempty"`
	FirstName              string   `json:"firstName,omitempty"`
	Gender                 string   `json:"gender,omitempty"`
	ID                     int      `json:"id,omitempty"`
	IgnoreDuplicates       bool     `json:"ignoreDuplicates,omitempty"`
	InactiveFlag           bool     `json:"inactiveFlag,omitempty"`
	IntegratorTags         []string `json:"integratorTags,omitempty"`
	LastName               string   `json:"lastName,omitempty"`
	LinkedInURL            string   `json:"linkedInUrl,omitempty"`
	ManagerContact         struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"managerContact,omitzero"`
	MarriedFlag bool   `json:"marriedFlag,omitempty"`
	MobileGUID  string `json:"mobileGuid,omitempty"`
	NickName    string `json:"nickName,omitempty"`
	Photo       struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"photo,omitzero"`
	PortalPassword      string `json:"portalPassword,omitempty"`
	PortalSecurityLevel int    `json:"portalSecurityLevel,omitempty"`
	Presence            string `json:"presence,omitempty"`
	Relationship        struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"relationship,omitzero"`
	RelationshipOverride string `json:"relationshipOverride,omitempty"`
	School               string `json:"school,omitempty"`
	SecurityIdentifier   string `json:"securityIdentifier,omitempty"`
	SignificantOther     string `json:"significantOther,omitempty"`
	Site                 struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"site,omitzero"`
	State      string `json:"state,omitempty"`
	Title      string `json:"title,omitempty"`
	TwitterURL string `json:"twitterUrl,omitempty"`
	TypeIds    []int  `json:"typeIds,omitempty"`
	Types      []struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"types,omitzero"`
	UnsubscribeFlag    bool   `json:"unsubscribeFlag,omitempty"`
	UserDefinedField1  string `json:"userDefinedField1,omitempty"`
	UserDefinedField10 string `json:"userDefinedField10,omitempty"`
	UserDefinedField2  string `json:"userDefinedField2,omitempty"`
	UserDefinedField3  string `json:"userDefinedField3,omitempty"`
	UserDefinedField4  string `json:"userDefinedField4,omitempty"`
	UserDefinedField5  string `json:"userDefinedField5,omitempty"`
	UserDefinedField6  string `json:"userDefinedField6,omitempty"`
	UserDefinedField7  string `json:"userDefinedField7,omitempty"`
	UserDefinedField8  string `json:"userDefinedField8,omitempty"`
	UserDefinedField9  string `json:"userDefinedField9,omitempty"`
	Zip                string `json:"zip,omitempty"`
}

type Member struct {
	Info                                     any     `json:"_info,omitempty"`
	AdminFlag                                bool    `json:"adminFlag,omitempty"`
	AgreementInvoicingDisplayOptions         string  `json:"agreementInvoicingDisplayOptions,omitempty"`
	AllowExpensesEnteredAgainstCompaniesFlag bool    `json:"allowExpensesEnteredAgainstCompaniesFlag,omitempty"`
	AllowInCellEntryOnTimeSheet              bool    `json:"allowInCellEntryOnTimeSheet,omitempty"`
	AuthenticationServiceType                string  `json:"authenticationServiceType,omitempty"`
	AutoPopupQuickNotesWithStopwatch         bool    `json:"autoPopupQuickNotesWithStopwatch,omitempty"`
	AutoStartStopwatch                       bool    `json:"autoStartStopwatch,omitempty"`
	BillableForecast                         float64 `json:"billableForecast,omitempty"`
	Calendar                                 struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"calendar,omitzero"`
	CalendarSyncIntegrationFlag bool   `json:"calendarSyncIntegrationFlag,omitempty"`
	ClientID                    string `json:"clientId,omitempty"`
	CompanyActivityTabFormat    string `json:"companyActivityTabFormat,omitempty"`
	CopyColumnLayoutsAndFilters bool   `json:"copyColumnLayoutsAndFilters,omitempty"`
	CopyPodLayouts              bool   `json:"copyPodLayouts,omitempty"`
	CopySharedDefaultViews      bool   `json:"copySharedDefaultViews,omitempty"`
	Country                     struct {
		Info       any    `json:"_info,omitempty"`
		ID         int    `json:"id,omitempty"`
		Identifier string `json:"identifier,omitempty"`
		Name       string `json:"name,omitempty"`
	} `json:"country,omitzero"`
	CustomFields []struct {
		Caption          string `json:"caption,omitempty"`
		ConnectWiseID    string `json:"connectWiseId,omitempty"`
		EntryMethod      string `json:"entryMethod,omitempty"`
		ID               int    `json:"id,omitempty"`
		NumberOfDecimals int    `json:"numberOfDecimals,omitempty"`
		Type             string `json:"type,omitempty"`
		Value            any    `json:"value,omitempty"`
	} `json:"customFields,omitzero"`
	DailyCapacity     float64 `json:"dailyCapacity,omitempty"`
	DaysTolerance     int     `json:"daysTolerance,omitempty"`
	DefaultDepartment struct {
		Info       any    `json:"_info,omitempty"`
		ID         int    `json:"id,omitempty"`
		Identifier string `json:"identifier,omitempty"`
		Name       string `json:"name,omitempty"`
	} `json:"defaultDepartment,omitzero"`
	DefaultEmail    string `json:"defaultEmail,omitempty"`
	DefaultLocation struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"defaultLocation,omitzero"`
	DefaultPhone    string `json:"defaultPhone,omitempty"`
	DirectionalSync struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"directionalSync,omitzero"`
	DisableOnlineFlag            bool   `json:"disableOnlineFlag,omitempty"`
	EmployeeIdentifer            string `json:"employeeIdentifer,omitempty"`
	EnableLdapAuthenticationFlag bool   `json:"enableLdapAuthenticationFlag,omitempty"`
	EnableMobileFlag             bool   `json:"enableMobileFlag,omitempty"`
	EnableMobileGpsFlag          bool   `json:"enableMobileGpsFlag,omitempty"`
	EnterTimeAgainstCompanyFlag  bool   `json:"enterTimeAgainstCompanyFlag,omitempty"`
	ExcludedProjectBoardIds      []int  `json:"excludedProjectBoardIds,omitempty"`
	ExcludedServiceBoardIds      []int  `json:"excludedServiceBoardIds,omitempty"`
	ExpenseApprover              struct {
		Info          any     `json:"_info,omitempty"`
		DailyCapacity float64 `json:"dailyCapacity,omitempty"`
		ID            int     `json:"id,omitempty"`
		Identifier    string  `json:"identifier,omitempty"`
		Name          string  `json:"name,omitempty"`
	} `json:"expenseApprover,omitzero"`
	FirstName                         string    `json:"firstName"`
	FromMemberRecID                   int       `json:"fromMemberRecId,omitempty"`
	FromMemberTemplateRecID           int       `json:"fromMemberTemplateRecId,omitempty"`
	GlobalSearchDefaultSort           string    `json:"globalSearchDefaultSort,omitempty"`
	GlobalSearchDefaultTicketFilter   string    `json:"globalSearchDefaultTicketFilter,omitempty"`
	HideMemberInDispatchPortalFlag    bool      `json:"hideMemberInDispatchPortalFlag,omitempty"`
	HireDate                          time.Time `json:"hireDate"`
	HomeEmail                         string    `json:"homeEmail,omitempty"`
	HomeExtension                     string    `json:"homeExtension,omitempty"`
	HomePhone                         string    `json:"homePhone,omitempty"`
	HourlyCost                        float64   `json:"hourlyCost,omitempty"`
	HourlyRate                        float64   `json:"hourlyRate,omitempty"`
	ID                                int       `json:"id,omitempty"`
	Identifier                        string    `json:"identifier"`
	InactiveDate                      time.Time `json:"inactiveDate,omitzero"`
	InactiveFlag                      bool      `json:"inactiveFlag,omitempty"`
	IncludeInUtilizationReportingFlag bool      `json:"includeInUtilizationReportingFlag,omitempty"`
	InvoiceScreenDefaultTabFormat     string    `json:"invoiceScreenDefaultTabFormat,omitempty"`
	InvoiceTimeTabFormat              string    `json:"invoiceTimeTabFormat,omitempty"`
	InvoicingDisplayOptions           string    `json:"invoicingDisplayOptions,omitempty"`
	LastLogin                         string    `json:"lastLogin,omitempty"`
	LastName                          string    `json:"lastName"`
	LdapConfiguration                 struct {
		Info   any    `json:"_info,omitempty"`
		ID     int    `json:"id,omitempty"`
		Name   string `json:"name,omitempty"`
		Server string `json:"server,omitempty"`
	} `json:"ldapConfiguration,omitzero"`
	LdapUserName    string  `json:"ldapUserName,omitempty"`
	LicenseClass    string  `json:"licenseClass,omitempty"`
	MapiName        string  `json:"mapiName,omitempty"`
	MemberPersonas  []int   `json:"memberPersonas,omitempty"`
	MiddleInitial   string  `json:"middleInitial,omitempty"`
	MinimumHours    float64 `json:"minimumHours,omitempty"`
	MobileEmail     string  `json:"mobileEmail,omitempty"`
	MobileExtension string  `json:"mobileExtension,omitempty"`
	MobilePhone     string  `json:"mobilePhone,omitempty"`
	Notes           string  `json:"notes,omitempty"`
	Office365       struct {
		ID   string `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"office365,omitzero"`
	OfficeEmail          string `json:"officeEmail,omitempty"`
	OfficeExtension      string `json:"officeExtension,omitempty"`
	OfficePhone          string `json:"officePhone,omitempty"`
	PartnerPortalFlag    bool   `json:"partnerPortalFlag,omitempty"`
	Password             string `json:"password,omitempty"`
	PhoneIntegrationType string `json:"phoneIntegrationType,omitempty"`
	PhoneSource          string `json:"phoneSource,omitempty"`
	Photo                struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"photo,omitzero"`
	PrimaryEmail        string `json:"primaryEmail,omitempty"`
	ProjectDefaultBoard struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"projectDefaultBoard,omitzero"`
	ProjectDefaultDepartment struct {
		Info       any    `json:"_info,omitempty"`
		ID         int    `json:"id,omitempty"`
		Identifier string `json:"identifier,omitempty"`
		Name       string `json:"name,omitempty"`
	} `json:"projectDefaultDepartment,omitzero"`
	ProjectDefaultLocation struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"projectDefaultLocation,omitzero"`
	ReportCard struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"reportCard,omitzero"`
	ReportsTo struct {
		Info          any     `json:"_info,omitempty"`
		DailyCapacity float64 `json:"dailyCapacity,omitempty"`
		ID            int     `json:"id,omitempty"`
		Identifier    string  `json:"identifier,omitempty"`
		Name          string  `json:"name,omitempty"`
	} `json:"reportsTo,omitzero"`
	RequireExpenseEntryFlag               bool `json:"requireExpenseEntryFlag,omitempty"`
	RequireStartAndEndTimeOnTimeEntryFlag bool `json:"requireStartAndEndTimeOnTimeEntryFlag,omitempty"`
	RequireTimeSheetEntryFlag             bool `json:"requireTimeSheetEntryFlag,omitempty"`
	RestrictDefaultSalesTerritoryFlag     bool `json:"restrictDefaultSalesTerritoryFlag,omitempty"`
	RestrictDefaultWarehouseBinFlag       bool `json:"restrictDefaultWarehouseBinFlag,omitempty"`
	RestrictDefaultWarehouseFlag          bool `json:"restrictDefaultWarehouseFlag,omitempty"`
	RestrictDepartmentFlag                bool `json:"restrictDepartmentFlag,omitempty"`
	RestrictLocationFlag                  bool `json:"restrictLocationFlag,omitempty"`
	RestrictProjectDefaultDepartmentFlag  bool `json:"restrictProjectDefaultDepartmentFlag,omitempty"`
	RestrictProjectDefaultLocationFlag    bool `json:"restrictProjectDefaultLocationFlag,omitempty"`
	RestrictScheduleFlag                  bool `json:"restrictScheduleFlag,omitempty"`
	RestrictServiceDefaultDepartmentFlag  bool `json:"restrictServiceDefaultDepartmentFlag,omitempty"`
	RestrictServiceDefaultLocationFlag    bool `json:"restrictServiceDefaultLocationFlag,omitempty"`
	SalesDefaultLocation                  struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"salesDefaultLocation,omitzero"`
	ScheduleCapacity          float64 `json:"scheduleCapacity,omitempty"`
	ScheduleDefaultDepartment struct {
		Info       any    `json:"_info,omitempty"`
		ID         int    `json:"id,omitempty"`
		Identifier string `json:"identifier,omitempty"`
		Name       string `json:"name,omitempty"`
	} `json:"scheduleDefaultDepartment,omitzero"`
	ScheduleDefaultLocation struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"scheduleDefaultLocation,omitzero"`
	SecurityLocation struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"securityLocation,omitzero"`
	SecurityRole struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"securityRole"`
	ServiceBoardTeamIds []int `json:"serviceBoardTeamIds,omitempty"`
	ServiceDefaultBoard struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"serviceDefaultBoard,omitzero"`
	ServiceDefaultDepartment struct {
		Info       any    `json:"_info,omitempty"`
		ID         int    `json:"id,omitempty"`
		Identifier string `json:"identifier,omitempty"`
		Name       string `json:"name,omitempty"`
	} `json:"serviceDefaultDepartment,omitzero"`
	ServiceDefaultLocation struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"serviceDefaultLocation,omitzero"`
	ServiceLocation struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"serviceLocation,omitzero"`
	Signature   string `json:"signature,omitempty"`
	SsoSettings struct {
		Info      any    `json:"_info,omitempty"`
		Email     string `json:"email,omitempty"`
		ID        int    `json:"id,omitempty"`
		SsoUserID string `json:"ssoUserId,omitempty"`
		UserName  string `json:"userName,omitempty"`
	} `json:"ssoSettings,omitzero"`
	StructureLevel struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"structureLevel,omitzero"`
	StsUserAdminURL string `json:"stsUserAdminUrl,omitempty"`
	Teams           []int  `json:"teams,omitempty"`
	TimeApprover    struct {
		Info          any     `json:"_info,omitempty"`
		DailyCapacity float64 `json:"dailyCapacity,omitempty"`
		ID            int     `json:"id,omitempty"`
		Identifier    string  `json:"identifier,omitempty"`
		Name          string  `json:"name,omitempty"`
	} `json:"timeApprover,omitzero"`
	TimeReminderEmailFlag bool   `json:"timeReminderEmailFlag,omitempty"`
	TimeSheetStartDate    string `json:"timeSheetStartDate,omitempty"`
	TimeZone              struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"timeZone,omitzero"`
	TimebasedOneTimePasswordActivated bool   `json:"timebasedOneTimePasswordActivated,omitempty"`
	Title                             string `json:"title,omitempty"`
	ToastNotificationFlag             bool   `json:"toastNotificationFlag,omitempty"`
	Token                             string `json:"token,omitempty"`
	Type                              struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"type,omitzero"`
	UseBrowserLanguageFlag bool   `json:"useBrowserLanguageFlag,omitempty"`
	VendorNumber           string `json:"vendorNumber,omitempty"`
	Warehouse              struct {
		Info       any    `json:"_info,omitempty"`
		ID         int    `json:"id,omitempty"`
		LockedFlag bool   `json:"lockedFlag,omitempty"`
		Name       string `json:"name,omitempty"`
	} `json:"warehouse,omitzero"`
	WarehouseBin struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"warehouseBin,omitzero"`
	WorkRole struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"workRole,omitzero"`
	WorkType struct {
		Info            any    `json:"_info,omitempty"`
		ID              int    `json:"id,omitempty"`
		Name            string `json:"name,omitempty"`
		UtilizationFlag bool   `json:"utilizationFlag,omitempty"`
	} `json:"workType,omitzero"`
}

type Ticket struct {
	Info struct {
		LastUpdated time.Time `json:"lastUpdated,omitzero"`
		UpdatedBy   string    `json:"updatedBy"`
		DateEntered time.Time `json:"dateEntered"`
		EnteredBy   string    `json:"enteredBy"`
	} `json:"_info,omitzero"`
	ActualHours  float64 `json:"actualHours,omitempty"`
	AddressLine1 string  `json:"addressLine1,omitempty"`
	AddressLine2 string  `json:"addressLine2,omitempty"`
	Agreement    struct {
		Info           any    `json:"_info,omitempty"`
		ChargeFirmFlag bool   `json:"chargeFirmFlag,omitempty"`
		ID             int    `json:"id,omitempty"`
		Name           string `json:"name,omitempty"`
		Type           string `json:"type,omitempty"`
	} `json:"agreement,omitzero"`
	AgreementType              string  `json:"agreementType,omitempty"`
	AllowAllClientsPortalView  bool    `json:"allowAllClientsPortalView,omitempty"`
	Approved                   bool    `json:"approved,omitempty"`
	AutomaticEmailCc           string  `json:"automaticEmailCc,omitempty"`
	AutomaticEmailCcFlag       bool    `json:"automaticEmailCcFlag,omitempty"`
	AutomaticEmailContactFlag  bool    `json:"automaticEmailContactFlag,omitempty"`
	AutomaticEmailResourceFlag bool    `json:"automaticEmailResourceFlag,omitempty"`
	BillExpenses               string  `json:"billExpenses,omitempty"`
	BillProducts               string  `json:"billProducts,omitempty"`
	BillTime                   string  `json:"billTime,omitempty"`
	BillingAmount              float64 `json:"billingAmount,omitempty"`
	BillingMethod              string  `json:"billingMethod,omitempty"`
	Board                      struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"board,omitzero"`
	BudgetHours float64   `json:"budgetHours,omitempty"`
	City        string    `json:"city,omitempty"`
	ClosedBy    string    `json:"closedBy,omitempty"`
	ClosedDate  time.Time `json:"closedDate,omitzero"`
	ClosedFlag  bool      `json:"closedFlag,omitempty"`
	Company     struct {
		Info       any    `json:"_info,omitempty"`
		ID         int    `json:"id,omitempty"`
		Identifier string `json:"identifier,omitempty"`
		Name       string `json:"name,omitempty"`
	} `json:"company"`
	Contact struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"contact,omitzero"`
	ContactEmailAddress   string `json:"contactEmailAddress,omitempty"`
	ContactEmailLookup    string `json:"contactEmailLookup,omitempty"`
	ContactName           string `json:"contactName,omitempty"`
	ContactPhoneExtension string `json:"contactPhoneExtension,omitempty"`
	ContactPhoneNumber    string `json:"contactPhoneNumber,omitempty"`
	Country               struct {
		Info       any    `json:"_info,omitempty"`
		ID         int    `json:"id,omitempty"`
		Identifier string `json:"identifier,omitempty"`
		Name       string `json:"name,omitempty"`
	} `json:"country,omitzero"`
	Currency struct {
		Info                    any    `json:"_info,omitempty"`
		CurrencyCode            string `json:"currencyCode,omitempty"`
		CurrencyIdentifier      string `json:"currencyIdentifier,omitempty"`
		DecimalSeparator        string `json:"decimalSeparator,omitempty"`
		DisplayIDFlag           bool   `json:"displayIdFlag,omitempty"`
		DisplaySymbolFlag       bool   `json:"displaySymbolFlag,omitempty"`
		ID                      int    `json:"id,omitempty"`
		Name                    string `json:"name,omitempty"`
		NegativeParenthesesFlag bool   `json:"negativeParenthesesFlag,omitempty"`
		NumberOfDecimals        int    `json:"numberOfDecimals,omitempty"`
		RightAlign              bool   `json:"rightAlign,omitempty"`
		Symbol                  string `json:"symbol,omitempty"`
		ThousandsSeparator      string `json:"thousandsSeparator,omitempty"`
	} `json:"currency,omitzero"`
	CustomFields []struct {
		Caption          string `json:"caption,omitempty"`
		ConnectWiseID    string `json:"connectWiseId,omitempty"`
		EntryMethod      string `json:"entryMethod,omitempty"`
		ID               int    `json:"id,omitempty"`
		NumberOfDecimals int    `json:"numberOfDecimals,omitempty"`
		Type             string `json:"type,omitempty"`
		Value            any    `json:"value,omitempty"`
	} `json:"customFields,omitzero"`
	CustomerUpdatedFlag bool   `json:"customerUpdatedFlag,omitempty"`
	DateResolved        string `json:"dateResolved,omitempty"`
	DateResplan         string `json:"dateResplan,omitempty"`
	DateResponded       string `json:"dateResponded,omitempty"`
	Department          struct {
		Info       any    `json:"_info,omitempty"`
		ID         int    `json:"id,omitempty"`
		Identifier string `json:"identifier,omitempty"`
		Name       string `json:"name,omitempty"`
	} `json:"department,omitzero"`
	Duration                 int       `json:"duration,omitempty"`
	EscalationLevel          int       `json:"escalationLevel,omitempty"`
	EscalationStartDateUTC   string    `json:"escalationStartDateUTC,omitempty"`
	EstimatedExpenseCost     float64   `json:"estimatedExpenseCost,omitempty"`
	EstimatedExpenseRevenue  float64   `json:"estimatedExpenseRevenue,omitempty"`
	EstimatedProductCost     float64   `json:"estimatedProductCost,omitempty"`
	EstimatedProductRevenue  float64   `json:"estimatedProductRevenue,omitempty"`
	EstimatedStartDate       time.Time `json:"estimatedStartDate,omitzero"`
	EstimatedTimeCost        float64   `json:"estimatedTimeCost,omitempty"`
	EstimatedTimeRevenue     float64   `json:"estimatedTimeRevenue,omitempty"`
	ExternalXRef             string    `json:"externalXRef,omitempty"`
	HasChildTicket           bool      `json:"hasChildTicket,omitempty"`
	HasMergedChildTicketFlag bool      `json:"hasMergedChildTicketFlag,omitempty"`
	HourlyRate               float64   `json:"hourlyRate,omitempty"`
	ID                       int       `json:"id,omitempty"`
	Impact                   string    `json:"impact,omitempty"`
	InitialDescription       string    `json:"initialDescription,omitempty"`
	InitialDescriptionFrom   string    `json:"initialDescriptionFrom,omitempty"`
	InitialInternalAnalysis  string    `json:"initialInternalAnalysis,omitempty"`
	InitialResolution        string    `json:"initialResolution,omitempty"`
	IntegratorTags           []string  `json:"integratorTags,omitempty"`
	IsInSLA                  bool      `json:"isInSla,omitempty"`
	Item                     struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"item,omitzero"`
	KnowledgeBaseCategoryID    int    `json:"knowledgeBaseCategoryId,omitempty"`
	KnowledgeBaseLinkID        int    `json:"knowledgeBaseLinkId,omitempty"`
	KnowledgeBaseLinkType      string `json:"knowledgeBaseLinkType,omitempty"`
	KnowledgeBaseSubCategoryID int    `json:"knowledgeBaseSubCategoryId,omitempty"`
	LagDays                    int    `json:"lagDays,omitempty"`
	LagNonworkingDaysFlag      bool   `json:"lagNonworkingDaysFlag,omitempty"`
	Location                   struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"location,omitzero"`
	MergedParentTicket struct {
		Info    any    `json:"_info,omitempty"`
		ID      int    `json:"id,omitempty"`
		Summary string `json:"summary,omitempty"`
	} `json:"mergedParentTicket,omitzero"`
	MinutesBeforeWaiting int    `json:"minutesBeforeWaiting,omitempty"`
	MinutesWaiting       int    `json:"minutesWaiting,omitempty"`
	MobileGUID           string `json:"mobileGuid,omitempty"`
	Opportunity          struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"opportunity,omitzero"`
	Owner struct {
		Info          any     `json:"_info,omitempty"`
		DailyCapacity float64 `json:"dailyCapacity,omitempty"`
		ID            int     `json:"id,omitempty"`
		Identifier    string  `json:"identifier,omitempty"`
		Name          string  `json:"name,omitempty"`
	} `json:"owner,omitzero"`
	ParentTicketID        int    `json:"parentTicketId,omitempty"`
	PoNumber              string `json:"poNumber,omitempty"`
	PredecessorClosedFlag bool   `json:"predecessorClosedFlag,omitempty"`
	PredecessorID         int    `json:"predecessorId,omitempty"`
	PredecessorType       string `json:"predecessorType,omitempty"`
	Priority              struct {
		Info  any    `json:"_info,omitempty"`
		ID    int    `json:"id,omitempty"`
		Level string `json:"level,omitempty"`
		Name  string `json:"name,omitempty"`
		Sort  int    `json:"sort,omitempty"`
	} `json:"priority,omitzero"`
	ProcessNotifications    bool      `json:"processNotifications,omitempty"`
	RecordType              string    `json:"recordType,omitempty"`
	RequestForChangeFlag    bool      `json:"requestForChangeFlag,omitempty"`
	RequiredDate            time.Time `json:"requiredDate,omitzero"`
	ResPlanMinutes          int       `json:"resPlanMinutes,omitempty"`
	ResolutionHours         float64   `json:"resolutionHours,omitempty"`
	ResolveMinutes          int       `json:"resolveMinutes,omitempty"`
	ResolvedBy              string    `json:"resolvedBy,omitempty"`
	Resources               string    `json:"resources,omitempty"`
	ResplanBy               string    `json:"resplanBy,omitempty"`
	ResplanHours            float64   `json:"resplanHours,omitempty"`
	ResplanSkippedMinutes   int       `json:"resplanSkippedMinutes,omitempty"`
	RespondMinutes          int       `json:"respondMinutes,omitempty"`
	RespondedBy             string    `json:"respondedBy,omitempty"`
	RespondedHours          float64   `json:"respondedHours,omitempty"`
	RespondedSkippedMinutes int       `json:"respondedSkippedMinutes,omitempty"`
	ServiceLocation         struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"serviceLocation,omitzero"`
	Severity string `json:"severity,omitempty"`
	Site     struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"site,omitzero"`
	SiteName     string `json:"siteName,omitempty"`
	SkipCallback bool   `json:"skipCallback,omitempty"`
	SLA          struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"sla,omitzero"`
	SLAStatus string `json:"slaStatus,omitempty"`
	Source    struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"source,omitzero"`
	StateIdentifier string `json:"stateIdentifier,omitempty"`
	Status          struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
		Sort int    `json:"sort,omitempty"`
	} `json:"status,omitzero"`
	SubBillingAmount float64 `json:"subBillingAmount,omitempty"`
	SubBillingMethod string  `json:"subBillingMethod,omitempty"`
	SubDateAccepted  string  `json:"subDateAccepted,omitempty"`
	SubType          struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"subType,omitzero"`
	Summary string `json:"summary"`
	Team    struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"team,omitzero"`
	Type struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"type,omitzero"`
	WorkRole struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"workRole,omitzero"`
	WorkType struct {
		Info            any    `json:"_info,omitempty"`
		ID              int    `json:"id,omitempty"`
		Name            string `json:"name,omitempty"`
		UtilizationFlag bool   `json:"utilizationFlag,omitempty"`
	} `json:"workType,omitzero"`
	Zip string `json:"zip,omitempty"`
}

type ServiceTicketNote struct {
	Info    any `json:"_info,omitempty"`
	Contact struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"contact,omitzero"`
	CreatedBy             string    `json:"createdBy,omitempty"`
	CustomerUpdatedFlag   bool      `json:"customerUpdatedFlag,omitempty"`
	DateCreated           time.Time `json:"dateCreated,omitzero"`
	DetailDescriptionFlag bool      `json:"detailDescriptionFlag,omitempty"`
	ExternalFlag          bool      `json:"externalFlag,omitempty"`
	ID                    int       `json:"id,omitempty"`
	InternalAnalysisFlag  bool      `json:"internalAnalysisFlag,omitempty"`
	InternalFlag          bool      `json:"internalFlag,omitempty"`
	IssueFlag             bool      `json:"issueFlag,omitempty"`
	Member                struct {
		Info          any     `json:"_info,omitempty"`
		DailyCapacity float64 `json:"dailyCapacity,omitempty"`
		ID            int     `json:"id,omitempty"`
		Identifier    string  `json:"identifier,omitempty"`
		Name          string  `json:"name,omitempty"`
	} `json:"member,omitzero"`
	ProcessNotifications bool    `json:"processNotifications,omitempty"`
	ResolutionFlag       bool    `json:"resolutionFlag,omitempty"`
	SentimentScore       float64 `json:"sentimentScore,omitempty"`
	Text                 string  `json:"text,omitempty"`
	TicketID             int     `json:"ticketId,omitempty"`
}

type ServiceTicketNoteAll struct {
	Info        any  `json:"_info,omitempty"`
	BundledFlag bool `json:"bundledFlag,omitempty"`
	Contact     struct {
		Info any    `json:"_info,omitempty"`
		ID   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"contact,omitzero"`
	CreatedByParentFlag   bool `json:"createdByParentFlag,omitempty"`
	DetailDescriptionFlag bool `json:"detailDescriptionFlag,omitempty"`
	ID                    int  `json:"id,omitempty"`
	InternalAnalysisFlag  bool `json:"internalAnalysisFlag,omitempty"`
	IsMarkdownFlag        bool `json:"isMarkdownFlag,omitempty"`
	IssueFlag             bool `json:"issueFlag,omitempty"`
	Member                struct {
		Info          any     `json:"_info,omitempty"`
		DailyCapacity float64 `json:"dailyCapacity,omitempty"`
		ID            int     `json:"id,omitempty"`
		Identifier    string  `json:"identifier,omitempty"`
		Name          string  `json:"name,omitempty"`
	} `json:"member,omitzero"`
	MergedFlag     bool   `json:"mergedFlag,omitempty"`
	NoteType       string `json:"noteType,omitempty"`
	OriginalAuthor string `json:"originalAuthor,omitempty"`
	ResolutionFlag bool   `json:"resolutionFlag,omitempty"`
	Text           string `json:"text,omitempty"`
	Ticket         struct {
		Info    any    `json:"_info,omitempty"`
		ID      int    `json:"id,omitempty"`
		Summary string `json:"summary,omitempty"`
	} `json:"ticket,omitzero"`
	TimeEnd   string `json:"timeEnd,omitempty"`
	TimeStart string `json:"timeStart,omitempty"`
}

type WebhookPayload struct {
	MessageID         string `json:"MessageID"`
	FromURL           string `json:"FromUrl"`
	CompanyID         string `json:"CompanyID"`
	MemberID          string `json:"MemberID"`
	Action            string `json:"Action"`
	Type              string `json:"Type"`
	ID                int    `json:"ID"`
	ProductInstanceID any    `json:"ProductInstanceID"`
	PartnerID         any    `json:"PartnerID"`
	Entity            string `json:"Entity"`
	Metadata          struct {
		KeyURL string `json:"key_url"`
	} `json:"Metadata"`
	CallbackObjectRecID int `json:"CallbackObjectRecID"`
}

package psa

import "time"

type Board struct {
	Info                           interface{} `json:"_info,omitempty"`
	AllSort                        string      `json:"allSort,omitempty"`
	AutoAssignLimitAmount          int         `json:"autoAssignLimitAmount,omitempty"`
	AutoAssignLimitFlag            bool        `json:"autoAssignLimitFlag,omitempty"`
	AutoAssignNewECTicketsFlag     bool        `json:"autoAssignNewECTicketsFlag,omitempty"`
	AutoAssignNewPortalTicketsFlag bool        `json:"autoAssignNewPortalTicketsFlag,omitempty"`
	AutoAssignNewTicketsFlag       bool        `json:"autoAssignNewTicketsFlag,omitempty"`
	AutoAssignTicketOwnerFlag      bool        `json:"autoAssignTicketOwnerFlag,omitempty"`
	AutoCloseStatus                struct {
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
		Sort int         `json:"sort,omitempty"`
	} `json:"autoCloseStatus,omitempty"`
	BillExpense                   string `json:"billExpense,omitempty"`
	BillProduct                   string `json:"billProduct,omitempty"`
	BillTicketSeparatelyFlag      bool   `json:"billTicketSeparatelyFlag,omitempty"`
	BillTicketsAfterClosedFlag    bool   `json:"billTicketsAfterClosedFlag,omitempty"`
	BillTime                      string `json:"billTime,omitempty"`
	BillUnapprovedTimeExpenseFlag bool   `json:"billUnapprovedTimeExpenseFlag,omitempty"`
	BoardIcon                     struct {
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
	} `json:"boardIcon,omitempty"`
	ClosedLoopAllFlag              bool `json:"closedLoopAllFlag,omitempty"`
	ClosedLoopDiscussionsFlag      bool `json:"closedLoopDiscussionsFlag,omitempty"`
	ClosedLoopInternalAnalysisFlag bool `json:"closedLoopInternalAnalysisFlag,omitempty"`
	ClosedLoopResolutionFlag       bool `json:"closedLoopResolutionFlag,omitempty"`
	ContactTemplate                struct {
		Info       interface{} `json:"_info,omitempty"`
		ID         int         `json:"id,omitempty"`
		Identifier string      `json:"identifier,omitempty"`
		Name       string      `json:"name,omitempty"`
		Type       string      `json:"type,omitempty"`
	} `json:"contactTemplate,omitempty"`
	Department struct {
		Info       interface{} `json:"_info,omitempty"`
		ID         int         `json:"id,omitempty"`
		Identifier string      `json:"identifier,omitempty"`
		Name       string      `json:"name,omitempty"`
	} `json:"department"`
	DiscussionsLockedFlag bool `json:"discussionsLockedFlag,omitempty"`
	DispatchMember        struct {
		Info          interface{} `json:"_info,omitempty"`
		DailyCapacity float64     `json:"dailyCapacity,omitempty"`
		ID            int         `json:"id,omitempty"`
		Identifier    string      `json:"identifier,omitempty"`
		Name          string      `json:"name,omitempty"`
	} `json:"dispatchMember,omitempty"`
	DutyManagerMember struct {
		Info          interface{} `json:"_info,omitempty"`
		DailyCapacity float64     `json:"dailyCapacity,omitempty"`
		ID            int         `json:"id,omitempty"`
		Identifier    string      `json:"identifier,omitempty"`
		Name          string      `json:"name,omitempty"`
	} `json:"dutyManagerMember,omitempty"`
	EmailConnectorAllowReopenClosedFlag       bool `json:"emailConnectorAllowReopenClosedFlag,omitempty"`
	EmailConnectorNeverReopenByDaysClosedFlag bool `json:"emailConnectorNeverReopenByDaysClosedFlag,omitempty"`
	EmailConnectorNeverReopenByDaysFlag       bool `json:"emailConnectorNeverReopenByDaysFlag,omitempty"`
	EmailConnectorNewTicketNoMatchFlag        bool `json:"emailConnectorNewTicketNoMatchFlag,omitempty"`
	EmailConnectorReopenDaysClosedLimit       int  `json:"emailConnectorReopenDaysClosedLimit,omitempty"`
	EmailConnectorReopenDaysLimit             int  `json:"emailConnectorReopenDaysLimit,omitempty"`
	EmailConnectorReopenResourcesFlag         bool `json:"emailConnectorReopenResourcesFlag,omitempty"`
	EmailConnectorReopenStatus                struct {
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
		Sort int         `json:"sort,omitempty"`
	} `json:"emailConnectorReopenStatus,omitempty"`
	ID                   int    `json:"id,omitempty"`
	InactiveFlag         bool   `json:"inactiveFlag,omitempty"`
	InternalAnalysisSort string `json:"internalAnalysisSort,omitempty"`
	Location             struct {
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
	} `json:"location"`
	MarkFirstNoteIssueFlag bool   `json:"markFirstNoteIssueFlag,omitempty"`
	Name                   string `json:"name"`
	NotifyEmailFrom        string `json:"notifyEmailFrom,omitempty"`
	NotifyEmailFromName    string `json:"notifyEmailFromName,omitempty"`
	OncallMember           struct {
		Info          interface{} `json:"_info,omitempty"`
		DailyCapacity float64     `json:"dailyCapacity,omitempty"`
		ID            int         `json:"id,omitempty"`
		Identifier    string      `json:"identifier,omitempty"`
		Name          string      `json:"name,omitempty"`
	} `json:"oncallMember,omitempty"`
	OverrideBillingSetupFlag bool   `json:"overrideBillingSetupFlag,omitempty"`
	PercentageCalculation    string `json:"percentageCalculation,omitempty"`
	ProblemSort              string `json:"problemSort,omitempty"`
	ProjectFlag              bool   `json:"projectFlag,omitempty"`
	ResolutionSort           string `json:"resolutionSort,omitempty"`
	ResourceTemplate         struct {
		Info       interface{} `json:"_info,omitempty"`
		ID         int         `json:"id,omitempty"`
		Identifier string      `json:"identifier,omitempty"`
		Name       string      `json:"name,omitempty"`
		Type       string      `json:"type,omitempty"`
	} `json:"resourceTemplate,omitempty"`
	RestrictBoardByDefaultFlag bool `json:"restrictBoardByDefaultFlag,omitempty"`
	SendToBundledFlag          bool `json:"sendToBundledFlag,omitempty"`
	SendToCCFlag               bool `json:"sendToCCFlag,omitempty"`
	SendToContactFlag          bool `json:"sendToContactFlag,omitempty"`
	SendToResourceFlag         bool `json:"sendToResourceFlag,omitempty"`
	ServiceManagerMember       struct {
		Info          interface{} `json:"_info,omitempty"`
		DailyCapacity float64     `json:"dailyCapacity,omitempty"`
		ID            int         `json:"id,omitempty"`
		Identifier    string      `json:"identifier,omitempty"`
		Name          string      `json:"name,omitempty"`
	} `json:"serviceManagerMember,omitempty"`
	ShowDependenciesFlag bool `json:"showDependenciesFlag,omitempty"`
	ShowEstimatesFlag    bool `json:"showEstimatesFlag,omitempty"`
	SignOffTemplate      struct {
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
	} `json:"signOffTemplate,omitempty"`
	TimeEntryDiscussionFlag       bool `json:"timeEntryDiscussionFlag,omitempty"`
	TimeEntryInternalAnalysisFlag bool `json:"timeEntryInternalAnalysisFlag,omitempty"`
	TimeEntryLockedFlag           bool `json:"timeEntryLockedFlag,omitempty"`
	TimeEntryResolutionFlag       bool `json:"timeEntryResolutionFlag,omitempty"`
	UseMemberDisplayNameFlag      bool `json:"useMemberDisplayNameFlag,omitempty"`
	WorkRole                      struct {
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
	} `json:"workRole,omitempty"`
	WorkType struct {
		Info            interface{} `json:"_info,omitempty"`
		ID              int         `json:"id,omitempty"`
		Name            string      `json:"name,omitempty"`
		UtilizationFlag bool        `json:"utilizationFlag,omitempty"`
	} `json:"workType,omitempty"`
}

type BoardStatus struct {
	Info  interface{} `json:"_info,omitempty"`
	Board struct {
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
	} `json:"board,omitempty"`
	ClosedStatus              bool   `json:"closedStatus,omitempty"`
	CustomStatusIndicatorName string `json:"customStatusIndicatorName,omitempty"`
	CustomerPortalDescription string `json:"customerPortalDescription,omitempty"`
	CustomerPortalFlag        bool   `json:"customerPortalFlag,omitempty"`
	DefaultFlag               bool   `json:"defaultFlag,omitempty"`
	DisplayOnBoard            bool   `json:"displayOnBoard,omitempty"`
	EmailTemplate             struct {
		Info       interface{} `json:"_info,omitempty"`
		ID         int         `json:"id,omitempty"`
		Identifier string      `json:"identifier,omitempty"`
		Name       string      `json:"name,omitempty"`
		Type       string      `json:"type,omitempty"`
	} `json:"emailTemplate,omitempty"`
	EscalationStatus   string `json:"escalationStatus,omitempty"`
	ID                 int    `json:"id,omitempty"`
	Inactive           bool   `json:"inactive,omitempty"`
	Name               string `json:"name"`
	RoundRobinCatchall bool   `json:"roundRobinCatchall,omitempty"`
	SaveTimeAsNote     bool   `json:"saveTimeAsNote,omitempty"`
	SortOrder          int    `json:"sortOrder,omitempty"`
	StatusIndicator    struct {
		Info       interface{} `json:"_info,omitempty"`
		ID         int         `json:"id,omitempty"`
		Identifier string      `json:"identifier,omitempty"`
		Name       string      `json:"name,omitempty"`
	} `json:"statusIndicator,omitempty"`
	TimeEntryNotAllowed bool `json:"timeEntryNotAllowed,omitempty"`
}

type Callback struct {
	Info                 interface{} `json:"_info,omitempty"`
	ConnectWiseID        string      `json:"connectWiseID,omitempty"`
	Description          string      `json:"description,omitempty"`
	ID                   int         `json:"id,omitempty"`
	InactiveFlag         bool        `json:"inactiveFlag,omitempty"`
	IsSelfSuppressedFlag bool        `json:"isSelfSuppressedFlag,omitempty"`
	IsSoapCallbackFlag   bool        `json:"isSoapCallbackFlag,omitempty"`
	Level                string      `json:"level"`
	MemberId             int         `json:"memberId,omitempty"`
	ObjectId             int         `json:"objectId,omitempty"`
	PayloadVersion       string      `json:"payloadVersion,omitempty"`
	Type                 string      `json:"type"`
	URL                  string      `json:"url"`
}

type Company struct {
	Id         int    `json:"id,omitempty"`
	Identifier string `json:"identifier"`
	Name       string `json:"name"`
	Status     struct {
		Id   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
		Info struct {
			AdditionalProp1 string `json:"additionalProp1,omitempty"`
			AdditionalProp2 string `json:"additionalProp2,omitempty"`
			AdditionalProp3 string `json:"additionalProp3,omitempty"`
		} `json:"_info,omitempty"`
	} `json:"status,omitempty"`
	AddressLine1 string `json:"addressLine1,omitempty"`
	AddressLine2 string `json:"addressLine2,omitempty"`
	City         string `json:"city,omitempty"`
	State        string `json:"state,omitempty"`
	Zip          string `json:"zip,omitempty"`
	Country      struct {
		Id         int    `json:"id,omitempty"`
		Identifier string `json:"identifier,omitempty"`
		Name       string `json:"name,omitempty"`
		Info       struct {
			AdditionalProp1 string `json:"additionalProp1,omitempty"`
			AdditionalProp2 string `json:"additionalProp2,omitempty"`
			AdditionalProp3 string `json:"additionalProp3,omitempty"`
		} `json:"_info,omitempty"`
	} `json:"country,omitempty"`
	PhoneNumber string `json:"phoneNumber,omitempty"`
	FaxNumber   string `json:"faxNumber,omitempty"`
	Website     string `json:"website,omitempty"`
	Territory   struct {
		Id   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
		Info struct {
			AdditionalProp1 string `json:"additionalProp1,omitempty"`
			AdditionalProp2 string `json:"additionalProp2,omitempty"`
			AdditionalProp3 string `json:"additionalProp3,omitempty"`
		} `json:"_info,omitempty"`
	} `json:"territory,omitempty"`
	Market struct {
		Id   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
		Info struct {
			AdditionalProp1 string `json:"additionalProp1,omitempty"`
			AdditionalProp2 string `json:"additionalProp2,omitempty"`
			AdditionalProp3 string `json:"additionalProp3,omitempty"`
		} `json:"_info,omitempty"`
	} `json:"market,omitempty"`
	AccountNumber  string `json:"accountNumber,omitempty"`
	DefaultContact struct {
		Id   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
		Info struct {
			AdditionalProp1 string `json:"additionalProp1,omitempty"`
			AdditionalProp2 string `json:"additionalProp2,omitempty"`
			AdditionalProp3 string `json:"additionalProp3,omitempty"`
		} `json:"_info,omitempty"`
	} `json:"defaultContact,omitempty"`
	DateAcquired time.Time `json:"dateAcquired,omitempty"`
	SicCode      struct {
		Id   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
		Info struct {
			AdditionalProp1 string `json:"additionalProp1,omitempty"`
			AdditionalProp2 string `json:"additionalProp2,omitempty"`
			AdditionalProp3 string `json:"additionalProp3,omitempty"`
		} `json:"_info,omitempty"`
	} `json:"sicCode,omitempty"`
	ParentCompany struct {
		Id         int    `json:"id,omitempty"`
		Identifier string `json:"identifier,omitempty"`
		Name       string `json:"name,omitempty"`
		Info       struct {
			AdditionalProp1 string `json:"additionalProp1,omitempty"`
			AdditionalProp2 string `json:"additionalProp2,omitempty"`
			AdditionalProp3 string `json:"additionalProp3,omitempty"`
		} `json:"_info,omitempty"`
	} `json:"parentCompany,omitempty"`
	AnnualRevenue     float64 `json:"annualRevenue,omitempty"`
	NumberOfEmployees int     `json:"numberOfEmployees,omitempty"`
	YearEstablished   int     `json:"yearEstablished,omitempty"`
	RevenueYear       int     `json:"revenueYear,omitempty"`
	OwnershipType     struct {
		Id   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
		Info struct {
			AdditionalProp1 string `json:"additionalProp1,omitempty"`
			AdditionalProp2 string `json:"additionalProp2,omitempty"`
			AdditionalProp3 string `json:"additionalProp3,omitempty"`
		} `json:"_info,omitempty"`
	} `json:"ownershipType,omitempty"`
	TimeZoneSetup struct {
		Id   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
		Info struct {
			AdditionalProp1 string `json:"additionalProp1,omitempty"`
			AdditionalProp2 string `json:"additionalProp2,omitempty"`
			AdditionalProp3 string `json:"additionalProp3,omitempty"`
		} `json:"_info,omitempty"`
	} `json:"timeZoneSetup,omitempty"`
	LeadSource      string `json:"leadSource,omitempty"`
	LeadFlag        bool   `json:"leadFlag,omitempty"`
	UnsubscribeFlag bool   `json:"unsubscribeFlag,omitempty"`
	Calendar        struct {
		Id   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
		Info struct {
			AdditionalProp1 string `json:"additionalProp1,omitempty"`
			AdditionalProp2 string `json:"additionalProp2,omitempty"`
			AdditionalProp3 string `json:"additionalProp3,omitempty"`
		} `json:"_info,omitempty"`
	} `json:"calendar,omitempty"`
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
		Id   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
		Info struct {
			AdditionalProp1 string `json:"additionalProp1,omitempty"`
			AdditionalProp2 string `json:"additionalProp2,omitempty"`
			AdditionalProp3 string `json:"additionalProp3,omitempty"`
		} `json:"_info,omitempty"`
	} `json:"taxCode,omitempty"`
	BillingTerms struct {
		Id   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
		Info struct {
			AdditionalProp1 string `json:"additionalProp1,omitempty"`
			AdditionalProp2 string `json:"additionalProp2,omitempty"`
			AdditionalProp3 string `json:"additionalProp3,omitempty"`
		} `json:"_info,omitempty"`
	} `json:"billingTerms,omitempty"`
	InvoiceTemplate struct {
		Id   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
		Info struct {
			AdditionalProp1 string `json:"additionalProp1,omitempty"`
			AdditionalProp2 string `json:"additionalProp2,omitempty"`
			AdditionalProp3 string `json:"additionalProp3,omitempty"`
		} `json:"_info,omitempty"`
	} `json:"invoiceTemplate,omitempty"`
	PricingSchedule struct {
		Id   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
		Info struct {
			AdditionalProp1 string `json:"additionalProp1,omitempty"`
			AdditionalProp2 string `json:"additionalProp2,omitempty"`
			AdditionalProp3 string `json:"additionalProp3,omitempty"`
		} `json:"_info,omitempty"`
	} `json:"pricingSchedule,omitempty"`
	CompanyEntityType struct {
		Id   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
		Info struct {
			AdditionalProp1 string `json:"additionalProp1,omitempty"`
			AdditionalProp2 string `json:"additionalProp2,omitempty"`
			AdditionalProp3 string `json:"additionalProp3,omitempty"`
		} `json:"_info,omitempty"`
	} `json:"companyEntityType,omitempty"`
	BillToCompany struct {
		Id         int    `json:"id,omitempty"`
		Identifier string `json:"identifier,omitempty"`
		Name       string `json:"name,omitempty"`
		Info       struct {
			AdditionalProp1 string `json:"additionalProp1,omitempty"`
			AdditionalProp2 string `json:"additionalProp2,omitempty"`
			AdditionalProp3 string `json:"additionalProp3,omitempty"`
		} `json:"_info,omitempty"`
	} `json:"billToCompany,omitempty"`
	BillingSite struct {
		Id   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
		Info struct {
			AdditionalProp1 string `json:"additionalProp1,omitempty"`
			AdditionalProp2 string `json:"additionalProp2,omitempty"`
			AdditionalProp3 string `json:"additionalProp3,omitempty"`
		} `json:"_info,omitempty"`
	} `json:"billingSite,omitempty"`
	BillingContact struct {
		Id   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
		Info struct {
			AdditionalProp1 string `json:"additionalProp1,omitempty"`
			AdditionalProp2 string `json:"additionalProp2,omitempty"`
			AdditionalProp3 string `json:"additionalProp3,omitempty"`
		} `json:"_info,omitempty"`
	} `json:"billingContact,omitempty"`
	InvoiceDeliveryMethod struct {
		Id   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
		Info struct {
			AdditionalProp1 string `json:"additionalProp1,omitempty"`
			AdditionalProp2 string `json:"additionalProp2,omitempty"`
			AdditionalProp3 string `json:"additionalProp3,omitempty"`
		} `json:"_info,omitempty"`
	} `json:"invoiceDeliveryMethod,omitempty"`
	InvoiceToEmailAddress string    `json:"invoiceToEmailAddress,omitempty"`
	InvoiceCCEmailAddress string    `json:"invoiceCCEmailAddress,omitempty"`
	DeletedFlag           bool      `json:"deletedFlag,omitempty"`
	DateDeleted           time.Time `json:"dateDeleted,omitempty"`
	DeletedBy             string    `json:"deletedBy,omitempty"`
	MobileGuid            string    `json:"mobileGuid,omitempty"`
	FacebookUrl           string    `json:"facebookUrl,omitempty"`
	TwitterUrl            string    `json:"twitterUrl,omitempty"`
	LinkedInUrl           string    `json:"linkedInUrl,omitempty"`
	Currency              struct {
		Id                      int    `json:"id,omitempty"`
		Symbol                  string `json:"symbol,omitempty"`
		CurrencyCode            string `json:"currencyCode,omitempty"`
		DecimalSeparator        string `json:"decimalSeparator,omitempty"`
		NumberOfDecimals        int    `json:"numberOfDecimals,omitempty"`
		ThousandsSeparator      string `json:"thousandsSeparator,omitempty"`
		NegativeParenthesesFlag bool   `json:"negativeParenthesesFlag,omitempty"`
		DisplaySymbolFlag       bool   `json:"displaySymbolFlag,omitempty"`
		CurrencyIdentifier      string `json:"currencyIdentifier,omitempty"`
		DisplayIdFlag           bool   `json:"displayIdFlag,omitempty"`
		RightAlign              bool   `json:"rightAlign,omitempty"`
		Name                    string `json:"name,omitempty"`
		Info                    struct {
			AdditionalProp1 string `json:"additionalProp1,omitempty"`
			AdditionalProp2 string `json:"additionalProp2,omitempty"`
			AdditionalProp3 string `json:"additionalProp3,omitempty"`
		} `json:"_info,omitempty"`
	} `json:"currency,omitempty"`
	TerritoryManager struct {
		Id            int    `json:"id,omitempty"`
		Identifier    string `json:"identifier,omitempty"`
		Name          string `json:"name,omitempty"`
		DailyCapacity int    `json:"dailyCapacity,omitempty"`
		Info          struct {
			AdditionalProp1 string `json:"additionalProp1,omitempty"`
			AdditionalProp2 string `json:"additionalProp2,omitempty"`
			AdditionalProp3 string `json:"additionalProp3,omitempty"`
		} `json:"_info,omitempty"`
	} `json:"territoryManager,omitempty"`
	ResellerIdentifier string `json:"resellerIdentifier,omitempty"`
	IsVendorFlag       bool   `json:"isVendorFlag,omitempty"`
	Types              []struct {
		Id   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
		Info struct {
			AdditionalProp1 string `json:"additionalProp1,omitempty"`
			AdditionalProp2 string `json:"additionalProp2,omitempty"`
			AdditionalProp3 string `json:"additionalProp3,omitempty"`
		} `json:"_info,omitempty"`
	} `json:"types,omitempty"`
	Site struct {
		Id   int    `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
		Info struct {
			AdditionalProp1 string `json:"additionalProp1,omitempty"`
			AdditionalProp2 string `json:"additionalProp2,omitempty"`
			AdditionalProp3 string `json:"additionalProp3,omitempty"`
		} `json:"_info,omitempty"`
	} `json:"site,omitempty"`
	IntegratorTags []string `json:"integratorTags,omitempty"`
	Info           struct {
		AdditionalProp1 string `json:"additionalProp1,omitempty"`
		AdditionalProp2 string `json:"additionalProp2,omitempty"`
		AdditionalProp3 string `json:"additionalProp3,omitempty"`
	} `json:"_info,omitempty"`
	CustomFields []struct {
		Id               int         `json:"id,omitempty"`
		Caption          string      `json:"caption,omitempty"`
		Type             string      `json:"type,omitempty"`
		EntryMethod      string      `json:"entryMethod,omitempty"`
		NumberOfDecimals int         `json:"numberOfDecimals,omitempty"`
		Value            interface{} `json:"value,omitempty"`
		ConnectWiseId    string      `json:"connectWiseId,omitempty"`
	} `json:"customFields,omitempty"`
}
type Contact struct {
	Info             interface{} `json:"_info,omitempty"`
	AddressLine1     string      `json:"addressLine1,omitempty"`
	AddressLine2     string      `json:"addressLine2,omitempty"`
	Anniversary      string      `json:"anniversary,omitempty"`
	AssistantContact struct {
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
	} `json:"assistantContact,omitempty"`
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
			Info interface{} `json:"_info,omitempty"`
			ID   int         `json:"id,omitempty"`
			Name string      `json:"name,omitempty"`
		} `json:"type,omitempty"`
		Value string `json:"value,omitempty"`
	} `json:"communicationItems,omitempty"`
	Company struct {
		Info       interface{} `json:"_info,omitempty"`
		ID         int         `json:"id,omitempty"`
		Identifier string      `json:"identifier,omitempty"`
		Name       string      `json:"name,omitempty"`
	} `json:"company,omitempty"`
	CompanyLocation struct {
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
	} `json:"companyLocation,omitempty"`
	Country struct {
		Info       interface{} `json:"_info,omitempty"`
		ID         int         `json:"id,omitempty"`
		Identifier string      `json:"identifier,omitempty"`
		Name       string      `json:"name,omitempty"`
	} `json:"country,omitempty"`
	CustomFields []struct {
		Caption          string      `json:"caption,omitempty"`
		ConnectWiseId    string      `json:"connectWiseId,omitempty"`
		EntryMethod      string      `json:"entryMethod,omitempty"`
		ID               int         `json:"id,omitempty"`
		NumberOfDecimals int         `json:"numberOfDecimals,omitempty"`
		Type             string      `json:"type,omitempty"`
		Value            interface{} `json:"value,omitempty"`
	} `json:"customFields,omitempty"`
	DefaultBillingFlag    bool   `json:"defaultBillingFlag,omitempty"`
	DefaultFlag           bool   `json:"defaultFlag,omitempty"`
	DefaultMergeContactId int    `json:"defaultMergeContactId,omitempty"`
	DefaultPhoneExtension string `json:"defaultPhoneExtension,omitempty"`
	DefaultPhoneNbr       string `json:"defaultPhoneNbr,omitempty"`
	DefaultPhoneType      string `json:"defaultPhoneType,omitempty"`
	Department            struct {
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
	} `json:"department,omitempty"`
	DisablePortalLoginFlag bool     `json:"disablePortalLoginFlag,omitempty"`
	FacebookUrl            string   `json:"facebookUrl,omitempty"`
	FirstName              string   `json:"firstName,omitempty"`
	Gender                 string   `json:"gender,omitempty"`
	ID                     int      `json:"id,omitempty"`
	IgnoreDuplicates       bool     `json:"ignoreDuplicates,omitempty"`
	InactiveFlag           bool     `json:"inactiveFlag,omitempty"`
	IntegratorTags         []string `json:"integratorTags,omitempty"`
	LastName               string   `json:"lastName,omitempty"`
	LinkedInUrl            string   `json:"linkedInUrl,omitempty"`
	ManagerContact         struct {
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
	} `json:"managerContact,omitempty"`
	MarriedFlag bool   `json:"marriedFlag,omitempty"`
	MobileGuid  string `json:"mobileGuid,omitempty"`
	NickName    string `json:"nickName,omitempty"`
	Photo       struct {
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
	} `json:"photo,omitempty"`
	PortalPassword      string `json:"portalPassword,omitempty"`
	PortalSecurityLevel int    `json:"portalSecurityLevel,omitempty"`
	Presence            string `json:"presence,omitempty"`
	Relationship        struct {
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
	} `json:"relationship,omitempty"`
	RelationshipOverride string `json:"relationshipOverride,omitempty"`
	School               string `json:"school,omitempty"`
	SecurityIdentifier   string `json:"securityIdentifier,omitempty"`
	SignificantOther     string `json:"significantOther,omitempty"`
	Site                 struct {
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
	} `json:"site,omitempty"`
	State      string `json:"state,omitempty"`
	Title      string `json:"title,omitempty"`
	TwitterUrl string `json:"twitterUrl,omitempty"`
	TypeIds    []int  `json:"typeIds,omitempty"`
	Types      []struct {
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
	} `json:"types,omitempty"`
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
	Info                                     interface{} `json:"_info,omitempty"`
	AdminFlag                                bool        `json:"adminFlag,omitempty"`
	AgreementInvoicingDisplayOptions         string      `json:"agreementInvoicingDisplayOptions,omitempty"`
	AllowExpensesEnteredAgainstCompaniesFlag bool        `json:"allowExpensesEnteredAgainstCompaniesFlag,omitempty"`
	AllowInCellEntryOnTimeSheet              bool        `json:"allowInCellEntryOnTimeSheet,omitempty"`
	AuthenticationServiceType                string      `json:"authenticationServiceType,omitempty"`
	AutoPopupQuickNotesWithStopwatch         bool        `json:"autoPopupQuickNotesWithStopwatch,omitempty"`
	AutoStartStopwatch                       bool        `json:"autoStartStopwatch,omitempty"`
	BillableForecast                         float64     `json:"billableForecast,omitempty"`
	Calendar                                 struct {
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
	} `json:"calendar,omitempty"`
	CalendarSyncIntegrationFlag bool   `json:"calendarSyncIntegrationFlag,omitempty"`
	ClientId                    string `json:"clientId,omitempty"`
	CompanyActivityTabFormat    string `json:"companyActivityTabFormat,omitempty"`
	CopyColumnLayoutsAndFilters bool   `json:"copyColumnLayoutsAndFilters,omitempty"`
	CopyPodLayouts              bool   `json:"copyPodLayouts,omitempty"`
	CopySharedDefaultViews      bool   `json:"copySharedDefaultViews,omitempty"`
	Country                     struct {
		Info       interface{} `json:"_info,omitempty"`
		ID         int         `json:"id,omitempty"`
		Identifier string      `json:"identifier,omitempty"`
		Name       string      `json:"name,omitempty"`
	} `json:"country,omitempty"`
	CustomFields []struct {
		Caption          string      `json:"caption,omitempty"`
		ConnectWiseId    string      `json:"connectWiseId,omitempty"`
		EntryMethod      string      `json:"entryMethod,omitempty"`
		ID               int         `json:"id,omitempty"`
		NumberOfDecimals int         `json:"numberOfDecimals,omitempty"`
		Type             string      `json:"type,omitempty"`
		Value            interface{} `json:"value,omitempty"`
	} `json:"customFields,omitempty"`
	DailyCapacity     float64 `json:"dailyCapacity,omitempty"`
	DaysTolerance     int     `json:"daysTolerance,omitempty"`
	DefaultDepartment struct {
		Info       interface{} `json:"_info,omitempty"`
		ID         int         `json:"id,omitempty"`
		Identifier string      `json:"identifier,omitempty"`
		Name       string      `json:"name,omitempty"`
	} `json:"defaultDepartment,omitempty"`
	DefaultEmail    string `json:"defaultEmail,omitempty"`
	DefaultLocation struct {
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
	} `json:"defaultLocation,omitempty"`
	DefaultPhone    string `json:"defaultPhone,omitempty"`
	DirectionalSync struct {
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
	} `json:"directionalSync,omitempty"`
	DisableOnlineFlag            bool   `json:"disableOnlineFlag,omitempty"`
	EmployeeIdentifer            string `json:"employeeIdentifer,omitempty"`
	EnableLdapAuthenticationFlag bool   `json:"enableLdapAuthenticationFlag,omitempty"`
	EnableMobileFlag             bool   `json:"enableMobileFlag,omitempty"`
	EnableMobileGpsFlag          bool   `json:"enableMobileGpsFlag,omitempty"`
	EnterTimeAgainstCompanyFlag  bool   `json:"enterTimeAgainstCompanyFlag,omitempty"`
	ExcludedProjectBoardIds      []int  `json:"excludedProjectBoardIds,omitempty"`
	ExcludedServiceBoardIds      []int  `json:"excludedServiceBoardIds,omitempty"`
	ExpenseApprover              struct {
		Info          interface{} `json:"_info,omitempty"`
		DailyCapacity float64     `json:"dailyCapacity,omitempty"`
		ID            int         `json:"id,omitempty"`
		Identifier    string      `json:"identifier,omitempty"`
		Name          string      `json:"name,omitempty"`
	} `json:"expenseApprover,omitempty"`
	FirstName                         string    `json:"firstName"`
	FromMemberRecId                   int       `json:"fromMemberRecId,omitempty"`
	FromMemberTemplateRecId           int       `json:"fromMemberTemplateRecId,omitempty"`
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
	InactiveDate                      time.Time `json:"inactiveDate,omitempty"`
	InactiveFlag                      bool      `json:"inactiveFlag,omitempty"`
	IncludeInUtilizationReportingFlag bool      `json:"includeInUtilizationReportingFlag,omitempty"`
	InvoiceScreenDefaultTabFormat     string    `json:"invoiceScreenDefaultTabFormat,omitempty"`
	InvoiceTimeTabFormat              string    `json:"invoiceTimeTabFormat,omitempty"`
	InvoicingDisplayOptions           string    `json:"invoicingDisplayOptions,omitempty"`
	LastLogin                         string    `json:"lastLogin,omitempty"`
	LastName                          string    `json:"lastName"`
	LdapConfiguration                 struct {
		Info   interface{} `json:"_info,omitempty"`
		ID     int         `json:"id,omitempty"`
		Name   string      `json:"name,omitempty"`
		Server string      `json:"server,omitempty"`
	} `json:"ldapConfiguration,omitempty"`
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
	} `json:"office365,omitempty"`
	OfficeEmail          string `json:"officeEmail,omitempty"`
	OfficeExtension      string `json:"officeExtension,omitempty"`
	OfficePhone          string `json:"officePhone,omitempty"`
	PartnerPortalFlag    bool   `json:"partnerPortalFlag,omitempty"`
	Password             string `json:"password,omitempty"`
	PhoneIntegrationType string `json:"phoneIntegrationType,omitempty"`
	PhoneSource          string `json:"phoneSource,omitempty"`
	Photo                struct {
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
	} `json:"photo,omitempty"`
	PrimaryEmail        string `json:"primaryEmail,omitempty"`
	ProjectDefaultBoard struct {
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
	} `json:"projectDefaultBoard,omitempty"`
	ProjectDefaultDepartment struct {
		Info       interface{} `json:"_info,omitempty"`
		ID         int         `json:"id,omitempty"`
		Identifier string      `json:"identifier,omitempty"`
		Name       string      `json:"name,omitempty"`
	} `json:"projectDefaultDepartment,omitempty"`
	ProjectDefaultLocation struct {
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
	} `json:"projectDefaultLocation,omitempty"`
	ReportCard struct {
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
	} `json:"reportCard,omitempty"`
	ReportsTo struct {
		Info          interface{} `json:"_info,omitempty"`
		DailyCapacity float64     `json:"dailyCapacity,omitempty"`
		ID            int         `json:"id,omitempty"`
		Identifier    string      `json:"identifier,omitempty"`
		Name          string      `json:"name,omitempty"`
	} `json:"reportsTo,omitempty"`
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
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
	} `json:"salesDefaultLocation,omitempty"`
	ScheduleCapacity          float64 `json:"scheduleCapacity,omitempty"`
	ScheduleDefaultDepartment struct {
		Info       interface{} `json:"_info,omitempty"`
		ID         int         `json:"id,omitempty"`
		Identifier string      `json:"identifier,omitempty"`
		Name       string      `json:"name,omitempty"`
	} `json:"scheduleDefaultDepartment,omitempty"`
	ScheduleDefaultLocation struct {
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
	} `json:"scheduleDefaultLocation,omitempty"`
	SecurityLocation struct {
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
	} `json:"securityLocation,omitempty"`
	SecurityRole struct {
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
	} `json:"securityRole"`
	ServiceBoardTeamIds []int `json:"serviceBoardTeamIds,omitempty"`
	ServiceDefaultBoard struct {
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
	} `json:"serviceDefaultBoard,omitempty"`
	ServiceDefaultDepartment struct {
		Info       interface{} `json:"_info,omitempty"`
		ID         int         `json:"id,omitempty"`
		Identifier string      `json:"identifier,omitempty"`
		Name       string      `json:"name,omitempty"`
	} `json:"serviceDefaultDepartment,omitempty"`
	ServiceDefaultLocation struct {
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
	} `json:"serviceDefaultLocation,omitempty"`
	ServiceLocation struct {
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
	} `json:"serviceLocation,omitempty"`
	Signature   string `json:"signature,omitempty"`
	SsoSettings struct {
		Info      interface{} `json:"_info,omitempty"`
		Email     string      `json:"email,omitempty"`
		ID        int         `json:"id,omitempty"`
		SsoUserId string      `json:"ssoUserId,omitempty"`
		UserName  string      `json:"userName,omitempty"`
	} `json:"ssoSettings,omitempty"`
	StructureLevel struct {
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
	} `json:"structureLevel,omitempty"`
	StsUserAdminUrl string `json:"stsUserAdminUrl,omitempty"`
	Teams           []int  `json:"teams,omitempty"`
	TimeApprover    struct {
		Info          interface{} `json:"_info,omitempty"`
		DailyCapacity float64     `json:"dailyCapacity,omitempty"`
		ID            int         `json:"id,omitempty"`
		Identifier    string      `json:"identifier,omitempty"`
		Name          string      `json:"name,omitempty"`
	} `json:"timeApprover,omitempty"`
	TimeReminderEmailFlag bool   `json:"timeReminderEmailFlag,omitempty"`
	TimeSheetStartDate    string `json:"timeSheetStartDate,omitempty"`
	TimeZone              struct {
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
	} `json:"timeZone,omitempty"`
	TimebasedOneTimePasswordActivated bool   `json:"timebasedOneTimePasswordActivated,omitempty"`
	Title                             string `json:"title,omitempty"`
	ToastNotificationFlag             bool   `json:"toastNotificationFlag,omitempty"`
	Token                             string `json:"token,omitempty"`
	Type                              struct {
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
	} `json:"type,omitempty"`
	UseBrowserLanguageFlag bool   `json:"useBrowserLanguageFlag,omitempty"`
	VendorNumber           string `json:"vendorNumber,omitempty"`
	Warehouse              struct {
		Info       interface{} `json:"_info,omitempty"`
		ID         int         `json:"id,omitempty"`
		LockedFlag bool        `json:"lockedFlag,omitempty"`
		Name       string      `json:"name,omitempty"`
	} `json:"warehouse,omitempty"`
	WarehouseBin struct {
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
	} `json:"warehouseBin,omitempty"`
	WorkRole struct {
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
	} `json:"workRole,omitempty"`
	WorkType struct {
		Info            interface{} `json:"_info,omitempty"`
		ID              int         `json:"id,omitempty"`
		Name            string      `json:"name,omitempty"`
		UtilizationFlag bool        `json:"utilizationFlag,omitempty"`
	} `json:"workType,omitempty"`
}

type Ticket struct {
	Info struct {
		LastUpdated time.Time `json:"lastUpdated,omitempty"`
		UpdatedBy   string    `json:"updatedBy"`
		DateEntered time.Time `json:"dateEntered"`
		EnteredBy   string    `json:"enteredBy"`
	} `json:"_info,omitempty"`
	ActualHours  float64 `json:"actualHours,omitempty"`
	AddressLine1 string  `json:"addressLine1,omitempty"`
	AddressLine2 string  `json:"addressLine2,omitempty"`
	Agreement    struct {
		Info           interface{} `json:"_info,omitempty"`
		ChargeFirmFlag bool        `json:"chargeFirmFlag,omitempty"`
		ID             int         `json:"id,omitempty"`
		Name           string      `json:"name,omitempty"`
		Type           string      `json:"type,omitempty"`
	} `json:"agreement,omitempty"`
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
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
	} `json:"board,omitempty"`
	BudgetHours float64   `json:"budgetHours,omitempty"`
	City        string    `json:"city,omitempty"`
	ClosedBy    string    `json:"closedBy,omitempty"`
	ClosedDate  time.Time `json:"closedDate,omitempty"`
	ClosedFlag  bool      `json:"closedFlag,omitempty"`
	Company     struct {
		Info       interface{} `json:"_info,omitempty"`
		ID         int         `json:"id,omitempty"`
		Identifier string      `json:"identifier,omitempty"`
		Name       string      `json:"name,omitempty"`
	} `json:"company"`
	Contact struct {
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
	} `json:"contact,omitempty"`
	ContactEmailAddress   string `json:"contactEmailAddress,omitempty"`
	ContactEmailLookup    string `json:"contactEmailLookup,omitempty"`
	ContactName           string `json:"contactName,omitempty"`
	ContactPhoneExtension string `json:"contactPhoneExtension,omitempty"`
	ContactPhoneNumber    string `json:"contactPhoneNumber,omitempty"`
	Country               struct {
		Info       interface{} `json:"_info,omitempty"`
		ID         int         `json:"id,omitempty"`
		Identifier string      `json:"identifier,omitempty"`
		Name       string      `json:"name,omitempty"`
	} `json:"country,omitempty"`
	Currency struct {
		Info                    interface{} `json:"_info,omitempty"`
		CurrencyCode            string      `json:"currencyCode,omitempty"`
		CurrencyIdentifier      string      `json:"currencyIdentifier,omitempty"`
		DecimalSeparator        string      `json:"decimalSeparator,omitempty"`
		DisplayIdFlag           bool        `json:"displayIdFlag,omitempty"`
		DisplaySymbolFlag       bool        `json:"displaySymbolFlag,omitempty"`
		ID                      int         `json:"id,omitempty"`
		Name                    string      `json:"name,omitempty"`
		NegativeParenthesesFlag bool        `json:"negativeParenthesesFlag,omitempty"`
		NumberOfDecimals        int         `json:"numberOfDecimals,omitempty"`
		RightAlign              bool        `json:"rightAlign,omitempty"`
		Symbol                  string      `json:"symbol,omitempty"`
		ThousandsSeparator      string      `json:"thousandsSeparator,omitempty"`
	} `json:"currency,omitempty"`
	CustomFields []struct {
		Caption          string      `json:"caption,omitempty"`
		ConnectWiseId    string      `json:"connectWiseId,omitempty"`
		EntryMethod      string      `json:"entryMethod,omitempty"`
		ID               int         `json:"id,omitempty"`
		NumberOfDecimals int         `json:"numberOfDecimals,omitempty"`
		Type             string      `json:"type,omitempty"`
		Value            interface{} `json:"value,omitempty"`
	} `json:"customFields,omitempty"`
	CustomerUpdatedFlag bool   `json:"customerUpdatedFlag,omitempty"`
	DateResolved        string `json:"dateResolved,omitempty"`
	DateResplan         string `json:"dateResplan,omitempty"`
	DateResponded       string `json:"dateResponded,omitempty"`
	Department          struct {
		Info       interface{} `json:"_info,omitempty"`
		ID         int         `json:"id,omitempty"`
		Identifier string      `json:"identifier,omitempty"`
		Name       string      `json:"name,omitempty"`
	} `json:"department,omitempty"`
	Duration                 int       `json:"duration,omitempty"`
	EscalationLevel          int       `json:"escalationLevel,omitempty"`
	EscalationStartDateUTC   string    `json:"escalationStartDateUTC,omitempty"`
	EstimatedExpenseCost     float64   `json:"estimatedExpenseCost,omitempty"`
	EstimatedExpenseRevenue  float64   `json:"estimatedExpenseRevenue,omitempty"`
	EstimatedProductCost     float64   `json:"estimatedProductCost,omitempty"`
	EstimatedProductRevenue  float64   `json:"estimatedProductRevenue,omitempty"`
	EstimatedStartDate       time.Time `json:"estimatedStartDate,omitempty"`
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
	IsInSla                  bool      `json:"isInSla,omitempty"`
	Item                     struct {
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
	} `json:"item,omitempty"`
	KnowledgeBaseCategoryId    int    `json:"knowledgeBaseCategoryId,omitempty"`
	KnowledgeBaseLinkId        int    `json:"knowledgeBaseLinkId,omitempty"`
	KnowledgeBaseLinkType      string `json:"knowledgeBaseLinkType,omitempty"`
	KnowledgeBaseSubCategoryId int    `json:"knowledgeBaseSubCategoryId,omitempty"`
	LagDays                    int    `json:"lagDays,omitempty"`
	LagNonworkingDaysFlag      bool   `json:"lagNonworkingDaysFlag,omitempty"`
	Location                   struct {
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
	} `json:"location,omitempty"`
	MergedParentTicket struct {
		Info    interface{} `json:"_info,omitempty"`
		ID      int         `json:"id,omitempty"`
		Summary string      `json:"summary,omitempty"`
	} `json:"mergedParentTicket,omitempty"`
	MinutesBeforeWaiting int    `json:"minutesBeforeWaiting,omitempty"`
	MinutesWaiting       int    `json:"minutesWaiting,omitempty"`
	MobileGuid           string `json:"mobileGuid,omitempty"`
	Opportunity          struct {
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
	} `json:"opportunity,omitempty"`
	Owner struct {
		Info          interface{} `json:"_info,omitempty"`
		DailyCapacity float64     `json:"dailyCapacity,omitempty"`
		ID            int         `json:"id,omitempty"`
		Identifier    string      `json:"identifier,omitempty"`
		Name          string      `json:"name,omitempty"`
	} `json:"owner,omitempty"`
	ParentTicketId        int    `json:"parentTicketId,omitempty"`
	PoNumber              string `json:"poNumber,omitempty"`
	PredecessorClosedFlag bool   `json:"predecessorClosedFlag,omitempty"`
	PredecessorId         int    `json:"predecessorId,omitempty"`
	PredecessorType       string `json:"predecessorType,omitempty"`
	Priority              struct {
		Info  interface{} `json:"_info,omitempty"`
		ID    int         `json:"id,omitempty"`
		Level string      `json:"level,omitempty"`
		Name  string      `json:"name,omitempty"`
		Sort  int         `json:"sort,omitempty"`
	} `json:"priority,omitempty"`
	ProcessNotifications    bool      `json:"processNotifications,omitempty"`
	RecordType              string    `json:"recordType,omitempty"`
	RequestForChangeFlag    bool      `json:"requestForChangeFlag,omitempty"`
	RequiredDate            time.Time `json:"requiredDate,omitempty"`
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
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
	} `json:"serviceLocation,omitempty"`
	Severity string `json:"severity,omitempty"`
	Site     struct {
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
	} `json:"site,omitempty"`
	SiteName     string `json:"siteName,omitempty"`
	SkipCallback bool   `json:"skipCallback,omitempty"`
	Sla          struct {
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
	} `json:"sla,omitempty"`
	SlaStatus string `json:"slaStatus,omitempty"`
	Source    struct {
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
	} `json:"source,omitempty"`
	StateIdentifier string `json:"stateIdentifier,omitempty"`
	Status          struct {
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
		Sort int         `json:"sort,omitempty"`
	} `json:"status,omitempty"`
	SubBillingAmount float64 `json:"subBillingAmount,omitempty"`
	SubBillingMethod string  `json:"subBillingMethod,omitempty"`
	SubDateAccepted  string  `json:"subDateAccepted,omitempty"`
	SubType          struct {
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
	} `json:"subType,omitempty"`
	Summary string `json:"summary"`
	Team    struct {
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
	} `json:"team,omitempty"`
	Type struct {
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
	} `json:"type,omitempty"`
	WorkRole struct {
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
	} `json:"workRole,omitempty"`
	WorkType struct {
		Info            interface{} `json:"_info,omitempty"`
		ID              int         `json:"id,omitempty"`
		Name            string      `json:"name,omitempty"`
		UtilizationFlag bool        `json:"utilizationFlag,omitempty"`
	} `json:"workType,omitempty"`
	Zip string `json:"zip,omitempty"`
}

type ServiceTicketNote struct {
	Info    interface{} `json:"_info,omitempty"`
	Contact struct {
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
	} `json:"contact,omitempty"`
	CreatedBy             string    `json:"createdBy,omitempty"`
	CustomerUpdatedFlag   bool      `json:"customerUpdatedFlag,omitempty"`
	DateCreated           time.Time `json:"dateCreated,omitempty"`
	DetailDescriptionFlag bool      `json:"detailDescriptionFlag,omitempty"`
	ExternalFlag          bool      `json:"externalFlag,omitempty"`
	ID                    int       `json:"id,omitempty"`
	InternalAnalysisFlag  bool      `json:"internalAnalysisFlag,omitempty"`
	InternalFlag          bool      `json:"internalFlag,omitempty"`
	IssueFlag             bool      `json:"issueFlag,omitempty"`
	Member                struct {
		Info          interface{} `json:"_info,omitempty"`
		DailyCapacity float64     `json:"dailyCapacity,omitempty"`
		ID            int         `json:"id,omitempty"`
		Identifier    string      `json:"identifier,omitempty"`
		Name          string      `json:"name,omitempty"`
	} `json:"member,omitempty"`
	ProcessNotifications bool    `json:"processNotifications,omitempty"`
	ResolutionFlag       bool    `json:"resolutionFlag,omitempty"`
	SentimentScore       float64 `json:"sentimentScore,omitempty"`
	Text                 string  `json:"text,omitempty"`
	TicketId             int     `json:"ticketId,omitempty"`
}

type ServiceTicketNoteAll struct {
	Info        interface{} `json:"_info,omitempty"`
	BundledFlag bool        `json:"bundledFlag,omitempty"`
	Contact     struct {
		Info interface{} `json:"_info,omitempty"`
		ID   int         `json:"id,omitempty"`
		Name string      `json:"name,omitempty"`
	} `json:"contact,omitempty"`
	CreatedByParentFlag   bool `json:"createdByParentFlag,omitempty"`
	DetailDescriptionFlag bool `json:"detailDescriptionFlag,omitempty"`
	ID                    int  `json:"id,omitempty"`
	InternalAnalysisFlag  bool `json:"internalAnalysisFlag,omitempty"`
	IsMarkdownFlag        bool `json:"isMarkdownFlag,omitempty"`
	IssueFlag             bool `json:"issueFlag,omitempty"`
	Member                struct {
		Info          interface{} `json:"_info,omitempty"`
		DailyCapacity float64     `json:"dailyCapacity,omitempty"`
		ID            int         `json:"id,omitempty"`
		Identifier    string      `json:"identifier,omitempty"`
		Name          string      `json:"name,omitempty"`
	} `json:"member,omitempty"`
	MergedFlag     bool   `json:"mergedFlag,omitempty"`
	NoteType       string `json:"noteType,omitempty"`
	OriginalAuthor string `json:"originalAuthor,omitempty"`
	ResolutionFlag bool   `json:"resolutionFlag,omitempty"`
	Text           string `json:"text,omitempty"`
	Ticket         struct {
		Info    interface{} `json:"_info,omitempty"`
		ID      int         `json:"id,omitempty"`
		Summary string      `json:"summary,omitempty"`
	} `json:"ticket,omitempty"`
	TimeEnd   string `json:"timeEnd,omitempty"`
	TimeStart string `json:"timeStart,omitempty"`
}

type WebhookPayload struct {
	MessageId         string      `json:"MessageId"`
	FromUrl           string      `json:"FromUrl"`
	CompanyId         string      `json:"CompanyId"`
	MemberId          string      `json:"MemberId"`
	Action            string      `json:"Action"`
	Type              string      `json:"Type"`
	ID                int         `json:"ID"`
	ProductInstanceId interface{} `json:"ProductInstanceId"`
	PartnerId         interface{} `json:"PartnerId"`
	Entity            string      `json:"Entity"`
	Metadata          struct {
		KeyUrl string `json:"key_url"`
	} `json:"Metadata"`
	CallbackObjectRecId int `json:"CallbackObjectRecId"`
}

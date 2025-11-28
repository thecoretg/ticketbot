package models

type AllRepos struct {
	APIKey        APIKeyRepository
	APIUser       APIUserRepository
	Config        ConfigRepository
	Notifications TicketNotificationRepository
	Forwards      UserForwardRepository
	NotifierRules NotifierRuleRepository
	WebexRoom     WebexRoomRepository
	CW            CWRepos
}

type CWRepos struct {
	Board   BoardRepository
	Company CompanyRepository
	Contact ContactRepository
	Member  MemberRepository
	Note    TicketNoteRepository
	Ticket  TicketRepository
}

package repos

type AllRepos struct {
	APIKey              APIKeyRepository
	APIUser             APIUserRepository
	Config              ConfigRepository
	Sessions            SessionRepository
	TicketNotifications TicketNotificationRepository
	NotifierForwards    NotifierForwardRepository
	NotifierRules       NotifierRuleRepository
	WebexRecipients     WebexRecipientRepository
	CW                  CWRepos
}

type CWRepos struct {
	Board        BoardRepository
	Company      CompanyRepository
	Contact      ContactRepository
	Member       MemberRepository
	Note         TicketNoteRepository
	Ticket       TicketRepository
	TicketStatus TicketStatusRepository
}

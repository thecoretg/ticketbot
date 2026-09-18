package repos

type AllRepos struct {
	APIKey              APIKeyRepository
	APIUser             APIUserRepository
	Config              ConfigRepository
	Logs                LogRepository
	Sessions            SessionRepository
	TOTPPending         TOTPPendingRepository
	TOTPRecovery        TOTPRecoveryRepository
	TicketNotifications TicketNotificationRepository
	NotifierForwards    NotifierForwardRepository
	WebexRecipients     WebexRecipientRepository
	TicketEvents        TicketEventRepository
	Workflows           WorkflowRepository
	Lists               ListRepository
	SSO                 SSOStore
	SSORoleMappings     SSORoleMappingRepository
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

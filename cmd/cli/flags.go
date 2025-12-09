package main

var (
	// general ID flag for commands that only need one
	id int

	cfgAttemptNotify bool
	cfgMaxMsgLen     int
	cfgMaxSyncs      int

	boardID     int
	recipientID int

	notifierID int

	forwardID        int
	forwardSrcID     int
	forwardDestID    int
	forwardStartDate string
	forwardEndDate   string
	forwardEnabled   bool
	forwardUserKeeps bool

	emailAddress string

	syncAll, syncBoards, syncWebexRecipients, syncTickets bool
	syncBoardIDs                                          []int
	maxConcurrentSyncs                                    int
)

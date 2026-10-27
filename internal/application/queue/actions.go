package queue

const (
	ActionQueueJoined        = "queue joined"
	ActionQueueLeft          = "queue left"
	ActionQueuePlaceSkipped  = "queue place skipped"
	ActionParticipantRemoved = "participant removed"
	ActionEventCreated       = "event created"
	ActionEventRenamed       = "event renamed"
	ActionEventClosed        = "event closed"
	ActionEventReopened      = "event reopened"
	ActionEventDeleted       = "event deleted"
	ActionEventCarriedOver   = "event carried over"
	ActionSwapRequested      = "swap requested"
	ActionSwapAccepted       = "swap accepted"
	ActionSwapDeclined       = "swap declined"
	ActionSwapCanceled       = "swap canceled"
	ActionInviteSent         = "invite sent"
	ActionInviteAccepted     = "invite accepted"
	ActionInviteDeclined     = "invite declined"
	ActionInviteCanceled     = "invite canceled"
)

const (
	AttrEventID     = "event_id"
	AttrFromEventID = "from_event_id"
	AttrToEventID   = "to_event_id"
	AttrTitle       = "title"
	AttrPosition    = "position"
	AttrCount       = "count"
	AttrPeerUserID  = "peer_user_id"
	AttrPeerName    = "peer_name"
)

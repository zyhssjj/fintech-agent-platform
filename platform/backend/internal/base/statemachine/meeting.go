package statemachine

const (
	MeetingDraft          = "draft"
	MeetingScheduled      = "scheduled"
	MeetingInProgress     = "in_progress"
	MeetingMinutesPending = "minutes_pending"
	MeetingCompleted      = "completed"
	MeetingCancelled      = "cancelled"
)

var meetingTransitions = map[string][]string{
	MeetingDraft:          {MeetingScheduled, MeetingCancelled},
	MeetingScheduled:      {MeetingInProgress, MeetingCancelled},
	MeetingInProgress:     {MeetingMinutesPending, MeetingCancelled},
	MeetingMinutesPending: {MeetingCompleted},
}

func CanTransitionMeeting(from, to string) bool {
	return canTransition(meetingTransitions, from, to)
}

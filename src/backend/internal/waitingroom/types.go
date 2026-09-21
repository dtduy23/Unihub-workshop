package waitingroom

// QueueStatus represents the result of a waiting room check
type QueueStatus int

const (
	QueueGranted       QueueStatus = 1 // User can proceed immediately
	QueueWaiting       QueueStatus = 2 // User has been placed in queue
	QueueAlreadyActive QueueStatus = 3 // User already has an active token
	QueueAlreadyQueued QueueStatus = 0 // User is already in queue
)

func (s QueueStatus) String() string {
	switch s {
	case QueueGranted:
		return "GRANTED"
	case QueueWaiting:
		return "WAITING"
	case QueueAlreadyActive:
		return "ALREADY_ACTIVE"
	case QueueAlreadyQueued:
		return "ALREADY_QUEUED"
	default:
		return "UNKNOWN"
	}
}

// WaitingRoomResult holds the response from the waiting room
type WaitingRoomResult struct {
	Status        QueueStatus `json:"status"`
	StatusText    string      `json:"status_text"`
	Position      int         `json:"position,omitempty"`       // Queue position (1-indexed), 0 if granted
	TotalInQueue  int         `json:"total_in_queue,omitempty"` // Total users waiting in queue
	RetryAfter    int         `json:"retry_after,omitempty"`    // Seconds until client should poll again
	AccessTTL     int         `json:"access_ttl,omitempty"`     // Seconds the access token is valid
	EstimatedWait int         `json:"estimated_wait,omitempty"` // Estimated seconds to wait
}

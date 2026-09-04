package port

// AgentMemoryResolveVO is intentionally small. The winner must be selected
// from the persisted conflict members; the service/repository validates that
// relationship again instead of trusting the request body.
type AgentMemoryResolveVO struct {
	WinnerAssertionID string `json:"winnerAssertionId" binding:"required"`
	Resolution        string `json:"resolution"`
}

type AgentMemoryRejectVO struct {
	Resolution string `json:"resolution"`
}

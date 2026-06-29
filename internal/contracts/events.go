package contracts

// EventsPayload carries security-oriented samples for v2 ingest kind=events.
type EventsPayload struct {
	ServerID         string              `json:"server_id"`
	Timestamp        float64             `json:"timestamp"`
	FailedLogins5Min *int                `json:"failed_logins_5min,omitempty"`
	SSHSessions      []SSHSessionSample  `json:"ssh_sessions,omitempty"`
	Events           []SecurityEvent     `json:"events,omitempty"`
}

type SecurityEvent struct {
	Type      string         `json:"type"`
	Timestamp float64        `json:"timestamp,omitempty"`
	Detail    map[string]any `json:"detail,omitempty"`
}

// HeartbeatPayload is the minimal liveness payload for kind=heartbeat.
type HeartbeatPayload struct {
	ServerID     string         `json:"server_id"`
	Timestamp    float64        `json:"timestamp"`
	AgentVersion string         `json:"agent_version"`
	AgentHealthy bool           `json:"agent_healthy"`
	Uptime       map[string]any `json:"uptime,omitempty"`
}

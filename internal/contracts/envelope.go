package contracts

const EnvelopeVersion = "2"

type IngestKind string

const (
	KindMetrics   IngestKind = "metrics"
	KindInventory IngestKind = "inventory"
	KindEvents    IngestKind = "events"
	KindHeartbeat IngestKind = "heartbeat"
	KindLogs      IngestKind = "logs"
)

// IngestEnvelope is the v2 wire format for POST /api/v1/monitoring/ingest/v2.
type IngestEnvelope struct {
	Version      string     `json:"version"`
	Kind         IngestKind `json:"kind"`
	ServerUUID   string     `json:"server_uuid"`
	ServerID     string     `json:"server_id"`
	Timestamp    float64    `json:"timestamp"`
	AgentVersion string     `json:"agent_version"`
	Payload      any        `json:"payload"`
}

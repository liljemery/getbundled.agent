package contracts

// LogLineEntry is one docker log line for ingest kind=logs.
type LogLineEntry struct {
	ToolInstallationUUID string `json:"tool_installation_uuid,omitempty"`
	DockerImage          string `json:"docker_image"`
	Stream               string `json:"stream,omitempty"`
	LoggedAt             string `json:"logged_at,omitempty"`
	Line                 string `json:"line"`
}

package contracts

// MetricsPayload mirrors backend app.monitoring.schemas.metrics.IngestPayload.
type MetricsPayload struct {
	ServerID         string                 `json:"server_id"`
	Timestamp        float64                `json:"timestamp,omitempty"`
	Hostname         string                 `json:"hostname,omitempty"`
	CPUUsagePercent  float64                `json:"cpu_usage_percent,omitempty"`
	Memory           map[string]any         `json:"memory,omitempty"`
	Disk             map[string]any         `json:"disk,omitempty"`
	Network          map[string]any         `json:"network,omitempty"`
	Services         []ServiceStatusSample  `json:"services,omitempty"`
	SSHSessions      []SSHSessionSample     `json:"ssh_sessions,omitempty"`
	FailedLogins5Min *int                   `json:"failed_logins_5min,omitempty"`
	Processes        []ProcessSample        `json:"processes,omitempty"`
	AgentVersion     string                 `json:"agent_version,omitempty"`
	AgentHealthy     bool                   `json:"agent_healthy,omitempty"`
	LoadAverage      map[string]any         `json:"load_average,omitempty"`
	Swap             map[string]any         `json:"swap,omitempty"`
	DiskIO           map[string]any         `json:"disk_io,omitempty"`
	Uptime           map[string]any         `json:"uptime,omitempty"`
	BootTime         map[string]any         `json:"boot_time,omitempty"`
	OpenPorts        map[string]any         `json:"open_ports,omitempty"`
	Docker           map[string]any         `json:"docker,omitempty"`
	FailedServices   map[string]any         `json:"failed_services,omitempty"`
}

type ProcessSample struct {
	PID            int32    `json:"pid,omitempty"`
	Name           string   `json:"name,omitempty"`
	CPUPercent     float64  `json:"cpu_percent,omitempty"`
	MemoryPercent  float64  `json:"memory_percent,omitempty"`
}

type ServiceStatusSample struct {
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

type SSHSessionSample struct {
	User      string  `json:"user,omitempty"`
	Host      string  `json:"host,omitempty"`
	StartedAt float64 `json:"started_at,omitempty"`
	Terminal  string  `json:"terminal,omitempty"`
}

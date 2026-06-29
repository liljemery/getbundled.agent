package contracts

// InventoryPayload is a slower-changing host snapshot for v2 ingest.
type InventoryPayload struct {
	ServerID   string            `json:"server_id"`
	Hostname   string            `json:"hostname,omitempty"`
	Timestamp  float64           `json:"timestamp"`
	OS         OSInventory       `json:"os"`
	Docker     *DockerInventory  `json:"docker,omitempty"`
	Containers []ContainerRecord `json:"containers,omitempty"`
	OpenPorts  []int             `json:"open_ports,omitempty"`
}

type OSInventory struct {
	Platform        string `json:"platform,omitempty"`
	PlatformFamily  string `json:"platform_family,omitempty"`
	PlatformVersion string `json:"platform_version,omitempty"`
	KernelVersion   string `json:"kernel_version,omitempty"`
	KernelArch      string `json:"kernel_arch,omitempty"`
	Hostname        string `json:"hostname,omitempty"`
}

type DockerInventory struct {
	Installed bool   `json:"installed"`
	Version   string `json:"version,omitempty"`
	Running   bool   `json:"running"`
}

type ContainerRecord struct {
	Name   string `json:"name,omitempty"`
	Status string `json:"status,omitempty"`
	Image  string `json:"image,omitempty"`
}

package collector

import (
	"encoding/json"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/getbundled/getbundled-agent/internal/config"
	"github.com/getbundled/getbundled-agent/internal/contracts"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

type Inventory struct {
	cfg config.Config
}

func NewInventory(cfg config.Config) *Inventory {
	return &Inventory{cfg: cfg}
}

func (i *Inventory) Collect() contracts.InventoryPayload {
	hostInfo, _ := host.Info()
	payload := contracts.InventoryPayload{
		ServerID:  i.cfg.ServerID,
		Hostname:  hostname(),
		Timestamp: float64(time.Now().Unix()),
		OS: contracts.OSInventory{
			Platform:        hostInfo.Platform,
			PlatformFamily:  hostInfo.PlatformFamily,
			PlatformVersion: hostInfo.PlatformVersion,
			KernelVersion:   hostInfo.KernelVersion,
			KernelArch:      runtime.GOARCH,
			Hostname:        hostInfo.Hostname,
		},
		Docker:     collectDockerInventory(),
		Containers: collectContainerRecords(),
		OpenPorts:  collectListeningPorts(100),
	}
	return payload
}

func collectDockerInventory() *contracts.DockerInventory {
	path, err := exec.LookPath("docker")
	if err != nil {
		return &contracts.DockerInventory{Installed: false, Running: false}
	}
	inv := &contracts.DockerInventory{Installed: true}
	out, err := exec.Command(path, "version", "--format", "{{.Server.Version}}").CombinedOutput()
	if err == nil {
		inv.Version = strings.TrimSpace(string(out))
	}
	inv.Running = serviceActive("docker") || processFallbackActive("docker")
	return inv
}

func collectContainerRecords() []contracts.ContainerRecord {
	raw := dockerContainers()
	if raw == nil {
		return nil
	}
	out := make([]contracts.ContainerRecord, 0, len(raw))
	for _, row := range raw {
		out = append(out, contracts.ContainerRecord{
			Name:   row["name"],
			Status: row["status"],
			Image:  row["image"],
		})
	}
	return out
}

func collectListeningPorts(limit int) []int {
	conns, err := net.Connections("inet")
	if err != nil {
		return nil
	}
	seen := make(map[int]struct{})
	var ports []int
	for _, c := range conns {
		if c.Status != "LISTEN" || c.Laddr.Port == 0 {
			continue
		}
		port := int(c.Laddr.Port)
		if _, ok := seen[port]; ok {
			continue
		}
		seen[port] = struct{}{}
		ports = append(ports, port)
		if len(ports) >= limit {
			break
		}
	}
	sort.Ints(ports)
	return ports
}

type criticalServiceSpec struct {
	Name  string `json:"name"`
	Kind  string `json:"kind"`
	Image string `json:"image"`
}

func loadCriticalServiceSpecs(configDir string) []criticalServiceSpec {
	path := configDir + "/critical_services.json"
	data, err := os.ReadFile(path)
	if err != nil {
		return defaultCriticalServices()
	}
	var payload struct {
		Services []criticalServiceSpec `json:"services"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return defaultCriticalServices()
	}
	var out []criticalServiceSpec
	for _, s := range payload.Services {
		name := strings.TrimSpace(s.Name)
		if name == "" {
			continue
		}
		out = append(out, s)
	}
	if len(out) == 0 {
		return defaultCriticalServices()
	}
	return out
}

func defaultCriticalServices() []criticalServiceSpec {
	return []criticalServiceSpec{{Name: "docker", Kind: "docker_engine"}}
}

func collectServices(configDir string) []contracts.ServiceStatusSample {
	specs := loadCriticalServiceSpecs(configDir)
	out := make([]contracts.ServiceStatusSample, 0, len(specs))
	for _, spec := range specs {
		kind := strings.ToLower(strings.TrimSpace(spec.Kind))
		if kind == "" {
			kind = "legacy"
		}
		var active bool
		switch kind {
		case "docker_engine":
			active = dockerEngineActive()
		case "docker_image":
			active = dockerImageActive(spec.Image)
		default:
			active = serviceActive(spec.Name) || processFallbackActive(spec.Name)
		}
		out = append(out, contracts.ServiceStatusSample{Name: spec.Name, Active: active})
	}
	return out
}

func dockerEngineActive() bool {
	if serviceActive("docker") {
		return true
	}
	return processFallbackActive("docker")
}

func dockerImageActive(image string) bool {
	if strings.TrimSpace(image) == "" {
		return false
	}
	if _, err := exec.LookPath("docker"); err != nil {
		return false
	}
	out, err := exec.Command("docker", "ps", "-a", "--filter", "ancestor="+image, "--format", "{{.ID}}").CombinedOutput()
	if err != nil {
		return false
	}
	id := strings.TrimSpace(string(out))
	if id == "" {
		out, err = exec.Command("docker", "ps", "-a", "--filter", "name="+image, "--format", "{{.ID}}").CombinedOutput()
		if err != nil {
			return false
		}
		id = strings.TrimSpace(string(out))
	}
	if id == "" {
		return false
	}
	state, err := exec.Command("docker", "inspect", "-f", "{{.State.Running}}", id).CombinedOutput()
	if err != nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(string(state)), "true")
}

func serviceActive(name string) bool {
	if runtime.GOOS == "linux" {
		if _, err := exec.LookPath("systemctl"); err != nil {
			return false
		}
		err := exec.Command("systemctl", "is-active", "--quiet", name).Run()
		return err == nil
	}
	return false
}

var processFallback = map[string][]string{
	"docker": {"dockerd", "containerd"},
	"mysql":  {"mysqld", "mariadbd"},
	"nginx":  {"nginx"},
	"redis":  {"redis-server", "redis"},
}

func processFallbackActive(name string) bool {
	needles, ok := processFallback[strings.ToLower(name)]
	if !ok {
		return false
	}
	procs, err := process.Processes()
	if err != nil {
		return false
	}
	for _, p := range procs {
		pname, _ := p.Name()
		pname = strings.ToLower(pname)
		for _, needle := range needles {
			if strings.Contains(pname, needle) {
				return true
			}
		}
	}
	return false
}

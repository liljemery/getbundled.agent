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
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

type Metrics struct {
	cfg config.Config
}

func NewMetrics(cfg config.Config) *Metrics {
	return &Metrics{cfg: cfg}
}

func (m *Metrics) Collect() contracts.MetricsPayload {
	now := float64(time.Now().Unix())
	cpuPct, _ := cpu.Percent(time.Second, false)
	var cpuUsage float64
	if len(cpuPct) > 0 {
		cpuUsage = cpuPct[0]
	}
	vm, _ := mem.VirtualMemory()
	root := "/"
	if runtime.GOOS == "windows" {
		root = os.Getenv("SystemDrive")
		if root == "" {
			root = "C:"
		}
		if !strings.HasSuffix(root, "\\") {
			root += "\\"
		}
	}
	du, _ := disk.Usage(root)
	netCounters, _ := net.IOCounters(false)
	services := collectServices(m.cfg.ConfigDir)
	failed := failedServiceNames(services)
	payload := contracts.MetricsPayload{
		ServerID:        m.cfg.ServerID,
		Timestamp:       now,
		Hostname:        hostname(),
		CPUUsagePercent: cpuUsage,
		Memory: map[string]any{
			"total":   vm.Total,
			"percent": vm.UsedPercent,
		},
		Disk: map[string]any{
			"percent": du.UsedPercent,
		},
		Services:       services,
		FailedServices: map[string]any{"names": failed},
		SSHSessions:    collectSSHSessions(),
		Processes:      collectTopProcesses(10),
		AgentVersion:   m.cfg.AgentVersion,
		AgentHealthy:   true,
		Uptime:         collectUptime(),
		BootTime:       collectBootTime(),
		OpenPorts:      collectOpenPorts(50),
		Docker:         collectDockerSummary(),
	}
	if netCounters != nil && len(netCounters) > 0 {
		payload.Network = map[string]any{
			"bytes_sent_total": netCounters[0].BytesSent,
			"bytes_recv_total": netCounters[0].BytesRecv,
		}
	}
	if runtime.GOOS != "windows" {
		if avg, err := load.Avg(); err == nil {
			payload.LoadAverage = map[string]any{"1m": avg.Load1, "5m": avg.Load5, "15m": avg.Load15}
		}
	}
	if swap, err := mem.SwapMemory(); err == nil {
		payload.Swap = map[string]any{"total": swap.Total, "used": swap.Used, "percent": swap.UsedPercent}
	}
	if dio, err := disk.IOCounters(); err == nil {
		var readBytes, writeBytes uint64
		for _, c := range dio {
			readBytes += c.ReadBytes
			writeBytes += c.WriteBytes
		}
		payload.DiskIO = map[string]any{"read_bytes": readBytes, "write_bytes": writeBytes}
	}
	if n := collectFailedLogins5Min(); n != nil {
		payload.FailedLogins5Min = n
	}
	return payload
}

func hostname() string {
	h, err := os.Hostname()
	if err != nil {
		return ""
	}
	return h
}

func collectUptime() map[string]any {
	boot, err := host.BootTime()
	if err != nil {
		return nil
	}
	return map[string]any{"seconds": float64(time.Now().Unix()) - float64(boot)}
}

func collectBootTime() map[string]any {
	boot, err := host.BootTime()
	if err != nil {
		return nil
	}
	return map[string]any{"unix": float64(boot)}
}

func collectOpenPorts(limit int) map[string]any {
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
	return map[string]any{"listening": ports}
}

func collectTopProcesses(limit int) []contracts.ProcessSample {
	procs, err := process.Processes()
	if err != nil {
		return nil
	}
	type ranked struct {
		sample contracts.ProcessSample
		cpu    float64
	}
	rankedList := make([]ranked, 0, len(procs))
	for _, p := range procs {
		name, _ := p.Name()
		cpuPct, _ := p.CPUPercent()
		memPct, _ := p.MemoryPercent()
		rankedList = append(rankedList, ranked{
			sample: contracts.ProcessSample{
				PID:           p.Pid,
				Name:          name,
				CPUPercent:     cpuPct,
				MemoryPercent: float64(memPct),
			},
			cpu: cpuPct,
		})
	}
	sort.Slice(rankedList, func(i, j int) bool { return rankedList[i].cpu > rankedList[j].cpu })
	if limit <= 0 || limit > len(rankedList) {
		limit = len(rankedList)
	}
	out := make([]contracts.ProcessSample, 0, limit)
	for i := 0; i < limit; i++ {
		out = append(out, rankedList[i].sample)
	}
	return out
}

func collectDockerSummary() map[string]any {
	containers := dockerContainers()
	if containers == nil {
		return nil
	}
	return map[string]any{"containers": containers}
}

func dockerContainers() []map[string]string {
	if _, err := exec.LookPath("docker"); err != nil {
		return nil
	}
	out, err := exec.Command("docker", "ps", "-a", "--format", "{{json .}}").CombinedOutput()
	if err != nil {
		return nil
	}
	var containers []map[string]string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var row map[string]string
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			continue
		}
		containers = append(containers, map[string]string{
			"name":   row["Names"],
			"status": row["Status"],
			"image":  row["Image"],
		})
		if len(containers) >= 30 {
			break
		}
	}
	return containers
}

func failedServiceNames(services []contracts.ServiceStatusSample) []string {
	var failed []string
	for _, s := range services {
		if !s.Active {
			failed = append(failed, s.Name)
		}
	}
	return failed
}

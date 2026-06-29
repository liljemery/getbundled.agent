package collector

import (
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/getbundled/getbundled-agent/internal/config"
	"github.com/getbundled/getbundled-agent/internal/contracts"
	"github.com/shirou/gopsutil/v3/host"
)

type Security struct {
	cfg config.Config
}

func NewSecurity(cfg config.Config) *Security {
	return &Security{cfg: cfg}
}

func (s *Security) Collect() contracts.EventsPayload {
	now := float64(time.Now().Unix())
	payload := contracts.EventsPayload{
		ServerID:    s.cfg.ServerID,
		Timestamp:   now,
		SSHSessions: collectSSHSessions(),
	}
	if n := collectFailedLogins5Min(); n != nil {
		payload.FailedLogins5Min = n
		payload.Events = []contracts.SecurityEvent{
			{
				Type:      "failed_logins_window",
				Timestamp: now,
				Detail:    map[string]any{"count_5min": *n},
			},
		}
	}
	return payload
}

func (s *Security) Heartbeat() contracts.HeartbeatPayload {
	return contracts.HeartbeatPayload{
		ServerID:     s.cfg.ServerID,
		Timestamp:    float64(time.Now().Unix()),
		AgentVersion: s.cfg.AgentVersion,
		AgentHealthy: true,
		Uptime:       collectUptime(),
	}
}

func collectSSHSessions() []contracts.SSHSessionSample {
	if runtime.GOOS == "windows" {
		return nil
	}
	users, err := host.Users()
	if err != nil {
		return nil
	}
	var sessions []contracts.SSHSessionSample
	for _, u := range users {
		tty := normalizeTTY(u.Terminal)
		if !strings.HasPrefix(tty, "pts/") {
			continue
		}
		sessions = append(sessions, contracts.SSHSessionSample{
			User:      u.User,
			Host:      u.Host,
			StartedAt: float64(u.Started),
			Terminal:  tty,
		})
	}
	return sessions
}

func normalizeTTY(terminal string) string {
	t := strings.TrimSpace(terminal)
	t = strings.TrimPrefix(t, "/dev/")
	return t
}

func collectFailedLogins5Min() *int {
	if runtime.GOOS == "windows" {
		return nil
	}
	if _, err := exec.LookPath("lastb"); err != nil {
		return nil
	}
	out, err := exec.Command("lastb", "-s", "-5min", "-F").CombinedOutput()
	if err != nil {
		if len(out) == 0 {
			return nil
		}
	}
	count := 0
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "btmp begins") {
			continue
		}
		count++
	}
	return &count
}

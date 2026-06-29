package collector

import (
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
	return contracts.EventsPayload{
		ServerID:    s.cfg.ServerID,
		Timestamp:   float64(time.Now().Unix()),
		SSHSessions: collectSSHSessions(),
	}
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

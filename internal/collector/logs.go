package collector

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/getbundled/getbundled-agent/internal/config"
	"github.com/getbundled/getbundled-agent/internal/contracts"
)

const (
	logTargetsFile      = "log_targets.json"
	logCursorsFile      = "log_cursors.json"
	logSinceWindow      = 20 * time.Second
	initialLogTailLines = 150
	maxLogBatch           = 200
)

type logTarget struct {
	ToolInstallationUUID string `json:"tool_installation_uuid"`
	DockerImage          string `json:"docker_image"`
	ComposeProject       string `json:"compose_project"`
	ComposeService       string `json:"compose_service"`
}

type logTargetsDoc struct {
	Targets []logTarget `json:"targets"`
}

type Logs struct {
	cfg     config.Config
	cursors map[string]string
}

func NewLogs(cfg config.Config) *Logs {
	return &Logs{cfg: cfg, cursors: loadLogCursors(cfg.ConfigDir)}
}

func (l *Logs) Collect() []contracts.LogLineEntry {
	if _, err := exec.LookPath("docker"); err != nil {
		return nil
	}
	targets := loadLogTargets(l.cfg.ConfigDir)
	if len(targets) == 0 {
		return nil
	}

	var entries []contracts.LogLineEntry
	for _, target := range targets {
		containerID := resolveLogContainerID(target)
		if containerID == "" {
			continue
		}
		cursorKey := logCursorKey(target, containerID)
		lastSeen := l.cursors[cursorKey]
		var lines []parsedDockerLog
		if lastSeen == "" {
			lines = fetchDockerLogsTail(containerID, initialLogTailLines)
		} else {
			lines = fetchDockerLogsSince(containerID, logSinceWindow)
		}
		emit := lastSeen == ""
		for _, line := range lines {
			if !emit {
				if line.raw == lastSeen {
					emit = true
					continue
				}
				continue
			}
			if line.message == "" {
				continue
			}
			entries = append(entries, contracts.LogLineEntry{
				ToolInstallationUUID: target.ToolInstallationUUID,
				DockerImage:          target.DockerImage,
				Stream:               line.stream,
				LoggedAt:             line.loggedAt,
				Line:                 line.message,
			})
			if len(entries) >= maxLogBatch {
				break
			}
		}
		if len(lines) > 0 {
			l.cursors[cursorKey] = lines[len(lines)-1].raw
		}
		if len(entries) >= maxLogBatch {
			break
		}
	}
	if len(entries) > 0 {
		saveLogCursors(l.cfg.ConfigDir, l.cursors)
	}
	return entries
}

func loadLogTargets(configDir string) []logTarget {
	path := filepath.Join(configDir, logTargetsFile)
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var doc logTargetsDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil
	}
	out := make([]logTarget, 0, len(doc.Targets))
	for _, target := range doc.Targets {
		if strings.TrimSpace(target.DockerImage) == "" {
			continue
		}
		out = append(out, target)
	}
	return out
}

func loadLogCursors(configDir string) map[string]string {
	path := filepath.Join(configDir, logCursorsFile)
	raw, err := os.ReadFile(path)
	if err != nil {
		return map[string]string{}
	}
	var cursors map[string]string
	if err := json.Unmarshal(raw, &cursors); err != nil || cursors == nil {
		return map[string]string{}
	}
	return cursors
}

func saveLogCursors(configDir string, cursors map[string]string) {
	path := filepath.Join(configDir, logCursorsFile)
	raw, err := json.Marshal(cursors)
	if err != nil {
		return
	}
	_ = os.WriteFile(path, raw, 0o644)
}

func logCursorKey(target logTarget, containerID string) string {
	inst := strings.TrimSpace(target.ToolInstallationUUID)
	if inst == "" {
		inst = target.DockerImage
	}
	return inst + "|" + containerID
}

func resolveLogContainerID(target logTarget) string {
	project := strings.TrimSpace(target.ComposeProject)
	service := strings.TrimSpace(target.ComposeService)
	if project != "" && service != "" {
		if id := dockerPSFilter(
			"label=com.docker.compose.project="+project,
			"label=com.docker.compose.service="+service,
		); id != "" {
			return id
		}
	}
	image := strings.TrimSpace(target.DockerImage)
	if image == "" {
		return ""
	}
	if id := dockerPSFilter("ancestor=" + image); id != "" {
		return id
	}
	return dockerPSFilter("name=" + strings.ReplaceAll(image, "/", "_"))
}

func dockerPSFilter(filters ...string) string {
	args := []string{"ps", "-q", "--filter", "status=running"}
	for _, filter := range filters {
		args = append(args, "--filter", filter)
	}
	out, err := exec.Command("docker", args...).Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}

type parsedDockerLog struct {
	raw      string
	loggedAt string
	stream   string
	message  string
}

func fetchDockerLogsSince(containerID string, since time.Duration) []parsedDockerLog {
	seconds := int(since.Seconds())
	if seconds < 1 {
		seconds = 1
	}
	out, err := exec.Command(
		"docker", "logs", "--timestamps", "--since", formatDurationSeconds(seconds), containerID,
	).CombinedOutput()
	if err != nil && len(out) == 0 {
		return nil
	}
	return parseDockerLogOutput(string(out))
}

func fetchDockerLogsTail(containerID string, tail int) []parsedDockerLog {
	if tail < 1 {
		tail = 1
	}
	out, err := exec.Command(
		"docker", "logs", "--timestamps", "--tail", fmt.Sprintf("%d", tail), containerID,
	).CombinedOutput()
	if err != nil && len(out) == 0 {
		return nil
	}
	return parseDockerLogOutput(string(out))
}

func formatDurationSeconds(seconds int) string {
	return fmt.Sprintf("%ds", seconds)
}

func parseDockerLogOutput(output string) []parsedDockerLog {
	var lines []parsedDockerLog
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		raw := scanner.Text()
		parsed := parseDockerLogLine(raw)
		if parsed.message == "" {
			continue
		}
		lines = append(lines, parsed)
	}
	return lines
}

func parseDockerLogLine(raw string) parsedDockerLog {
	line := strings.TrimSpace(raw)
	parsed := parsedDockerLog{raw: raw, stream: "stdout"}
	if line == "" {
		return parsed
	}
	if strings.HasPrefix(line, "stderr ") || strings.HasPrefix(line, "STDERR ") {
		parsed.stream = "stderr"
		line = strings.TrimSpace(line[6:])
	}
	space := strings.IndexByte(line, ' ')
	if space <= 0 {
		parsed.message = line
		return parsed
	}
	ts := strings.TrimSpace(line[:space])
	if strings.Contains(ts, "T") || strings.Contains(ts, "-") {
		parsed.loggedAt = normalizeDockerTimestamp(ts)
		parsed.message = strings.TrimSpace(line[space+1:])
		return parsed
	}
	parsed.message = line
	return parsed
}

func normalizeDockerTimestamp(ts string) string {
	if strings.HasSuffix(ts, "Z") {
		return ts
	}
	if strings.Contains(ts, "+") || strings.Count(ts, "-") > 2 {
		return ts
	}
	return ts + "Z"
}

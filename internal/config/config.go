package config

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

const AgentVersion = "2.2.1"

const (
	DefaultConfigDir       = "/opt/server-monitor"
	DefaultQueuePath       = "/opt/server-monitor/offline_queue.jsonl"
	DefaultIngestPath      = "/api/v1/monitoring/ingest/v2"
	HeartbeatInterval      = 10 * time.Second
	MetricsInterval        = 20 * time.Second
	InventoryInterval      = 10 * time.Minute
	SecurityInterval       = 3 * time.Minute
	LogsInterval           = 15 * time.Second
	QueueFlushInterval     = 30 * time.Second
)

type Config struct {
	ServerUUID   string
	AgentToken   string
	IngestURL    string
	ServerID     string
	ConfigDir    string
	QueuePath    string
	AgentVersion string
}

func Load() (Config, error) {
	cfg := Config{
		ServerUUID:   strings.TrimSpace(os.Getenv("GETBUNDLED_SERVER_UUID")),
		AgentToken:   strings.TrimSpace(os.Getenv("GETBUNDLED_AGENT_TOKEN")),
		IngestURL:    strings.TrimSpace(os.Getenv("GETBUNDLED_INGEST_BASE_URL")),
		ServerID:     strings.TrimSpace(os.Getenv("GETBUNDLED_SERVER_ID")),
		ConfigDir:    strings.TrimSpace(os.Getenv("GETBUNDLED_CONFIG_DIR")),
		QueuePath:    strings.TrimSpace(os.Getenv("GETBUNDLED_QUEUE_PATH")),
		AgentVersion: AgentVersion,
	}
	if cfg.ConfigDir == "" {
		cfg.ConfigDir = DefaultConfigDir
	}
	if cfg.QueuePath == "" {
		cfg.QueuePath = DefaultQueuePath
	}
	if cfg.IngestURL == "" {
		return cfg, fmt.Errorf("GETBUNDLED_INGEST_BASE_URL is required")
	}
	cfg.IngestURL = strings.TrimRight(cfg.IngestURL, "/") + DefaultIngestPath
	if cfg.ServerUUID == "" {
		return cfg, fmt.Errorf("GETBUNDLED_SERVER_UUID is required")
	}
	if cfg.AgentToken == "" {
		return cfg, fmt.Errorf("GETBUNDLED_AGENT_TOKEN is required")
	}
	if cfg.ServerID == "" {
		ip, err := primaryIPv4()
		if err != nil {
			return cfg, fmt.Errorf("GETBUNDLED_SERVER_ID unset and ipv4 fallback failed: %w", err)
		}
		cfg.ServerID = ip
	}
	return cfg, nil
}

func primaryIPv4() (string, error) {
	conn, err := net.Dial("udp4", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()
	addr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok || addr.IP == nil {
		return "", fmt.Errorf("no local ipv4 address")
	}
	return addr.IP.String(), nil
}

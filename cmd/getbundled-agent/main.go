package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/getbundled/getbundled-agent/internal/collector"
	"github.com/getbundled/getbundled-agent/internal/config"
	"github.com/getbundled/getbundled-agent/internal/contracts"
	"github.com/getbundled/getbundled-agent/internal/queue"
	"github.com/getbundled/getbundled-agent/internal/sender"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	q := queue.New(cfg.QueuePath)
	client := sender.New(cfg, q)
	metricsCollector := collector.NewMetrics(cfg)
	inventoryCollector := collector.NewInventory(cfg)
	securityCollector := collector.NewSecurity(cfg)

	log.Printf("getbundled-agent v%s server_uuid=%s server_id=%s", cfg.AgentVersion, cfg.ServerUUID, cfg.ServerID)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	heartbeatTick := time.NewTicker(config.HeartbeatInterval)
	metricsTick := time.NewTicker(config.MetricsInterval)
	inventoryTick := time.NewTicker(config.InventoryInterval)
	securityTick := time.NewTicker(config.SecurityInterval)
	queueTick := time.NewTicker(config.QueueFlushInterval)
	defer heartbeatTick.Stop()
	defer metricsTick.Stop()
	defer inventoryTick.Stop()
	defer securityTick.Stop()
	defer queueTick.Stop()

	send := func(kind contracts.IngestKind, payload any) {
		if err := client.Send(kind, payload); err != nil {
			log.Printf("send %s: %v", kind, err)
		}
	}

	sendHeartbeat := func() { send(contracts.KindHeartbeat, securityCollector.Heartbeat()) }
	sendMetrics := func() { send(contracts.KindMetrics, metricsCollector.Collect()) }
	sendInventory := func() { send(contracts.KindInventory, inventoryCollector.Collect()) }
	sendSecurity := func() { send(contracts.KindEvents, securityCollector.Collect()) }
	flushQueue := func() {
		if err := client.FlushQueue(20); err != nil {
			log.Printf("queue flush: %v", err)
		}
	}

	sendHeartbeat()
	sendMetrics()

	for {
		select {
		case <-stop:
			log.Printf("shutdown")
			return
		case <-heartbeatTick.C:
			sendHeartbeat()
		case <-metricsTick.C:
			sendMetrics()
		case <-inventoryTick.C:
			sendInventory()
		case <-securityTick.C:
			sendSecurity()
		case <-queueTick.C:
			flushQueue()
		}
	}
}

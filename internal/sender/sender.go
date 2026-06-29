package sender

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/getbundled/getbundled-agent/internal/auth"
	"github.com/getbundled/getbundled-agent/internal/config"
	"github.com/getbundled/getbundled-agent/internal/contracts"
	"github.com/getbundled/getbundled-agent/internal/queue"
)

type Client struct {
	cfg    config.Config
	http   *http.Client
	queue  *queue.Store
}

func New(cfg config.Config, q *queue.Store) *Client {
	return &Client{
		cfg:   cfg,
		queue: q,
		http: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *Client) Send(kind contracts.IngestKind, payload any) error {
	body, err := json.Marshal(contracts.IngestEnvelope{
		Version:      contracts.EnvelopeVersion,
		Kind:         kind,
		ServerUUID:   c.cfg.ServerUUID,
		ServerID:     c.cfg.ServerID,
		Timestamp:    float64(time.Now().Unix()),
		AgentVersion: c.cfg.AgentVersion,
		Payload:      payload,
	})
	if err != nil {
		return err
	}
	return c.postWithRetry(string(kind), body)
}

func (c *Client) FlushQueue(limit int) error {
	entries, err := c.queue.Drain(limit)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return nil
	}
	var acked []int64
	for _, entry := range entries {
		if err := c.postOnce(entry.Body); err != nil {
			break
		}
		acked = append(acked, entry.ID)
	}
	if len(acked) > 0 {
		return c.queue.Ack(acked)
	}
	return nil
}

func (c *Client) postWithRetry(kind string, body []byte) error {
	delays := []time.Duration{0, 2 * time.Second, 5 * time.Second}
	var lastErr error
	for _, delay := range delays {
		if delay > 0 {
			time.Sleep(delay)
		}
		if err := c.postOnce(body); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	if err := c.queue.Enqueue(kind, body); err != nil {
		return fmt.Errorf("post failed (%v) and queue enqueue failed: %w", lastErr, err)
	}
	return lastErr
}

func (c *Client) postOnce(body []byte) error {
	signed, err := auth.Sign(c.cfg.AgentToken, c.cfg.ServerUUID, body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, c.cfg.IngestURL, bytes.NewReader(signed.Body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.cfg.AgentToken)
	req.Header.Set(auth.HeaderTimestamp, signed.Timestamp)
	req.Header.Set(auth.HeaderNonce, signed.Nonce)
	req.Header.Set(auth.HeaderSignature, signed.Signature)
	req.Header.Set(auth.HeaderServerUUID, c.cfg.ServerUUID)

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		slurp, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("ingest status %d: %s", resp.StatusCode, string(slurp))
	}
	return nil
}

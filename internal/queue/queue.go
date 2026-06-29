package queue

import (
	"bufio"
	"encoding/json"
	"os"
	"sync"
	"time"
)

const (
	maxBytes    = 10 * 1024 * 1024
	maxAge      = 24 * time.Hour
)

type Entry struct {
	ID        int64           `json:"id"`
	Kind      string          `json:"kind"`
	Body      json.RawMessage `json:"body"`
	CreatedAt float64         `json:"created_at"`
}

type Store struct {
	path string
	mu   sync.Mutex
	next int64
}

func New(path string) *Store {
	return &Store{path: path, next: 1}
}

func (s *Store) Enqueue(kind string, body []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.pruneLocked(); err != nil {
		return err
	}
	entry := Entry{
		ID:        s.next,
		Kind:      kind,
		Body:      append(json.RawMessage(nil), body...),
		CreatedAt: float64(time.Now().Unix()),
	}
	s.next++
	f, err := os.OpenFile(s.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	return enc.Encode(entry)
}

func (s *Store) Drain(limit int) ([]Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := s.readAllLocked()
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, nil
	}
	if limit <= 0 || limit > len(entries) {
		limit = len(entries)
	}
	return entries[:limit], nil
}

func (s *Store) Ack(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	remove := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		remove[id] = struct{}{}
	}
	entries, err := s.readAllLocked()
	if err != nil {
		return err
	}
	kept := entries[:0]
	var maxID int64
	for _, e := range entries {
		if _, drop := remove[e.ID]; drop {
			continue
		}
		kept = append(kept, e)
		if e.ID > maxID {
			maxID = e.ID
		}
	}
	s.next = maxID + 1
	return s.rewriteLocked(kept)
}

func (s *Store) readAllLocked() ([]Entry, error) {
	f, err := os.Open(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	var entries []Entry
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		var e Entry
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			continue
		}
		entries = append(entries, e)
		if e.ID >= s.next {
			s.next = e.ID + 1
		}
	}
	return entries, sc.Err()
}

func (s *Store) pruneLocked() error {
	entries, err := s.readAllLocked()
	if err != nil {
		return err
	}
	cutoff := float64(time.Now().Add(-maxAge).Unix())
	kept := entries[:0]
	total := 0
	for _, e := range entries {
		if e.CreatedAt < cutoff {
			continue
		}
		kept = append(kept, e)
		total += len(e.Body) + 64
	}
	for total > maxBytes && len(kept) > 0 {
		total -= len(kept[0].Body) + 64
		kept = kept[1:]
	}
	return s.rewriteLocked(kept)
}

func (s *Store) rewriteLocked(entries []Entry) error {
	tmp := s.path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	for _, e := range entries {
		if err := enc.Encode(e); err != nil {
			_ = f.Close()
			_ = os.Remove(tmp)
			return err
		}
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if len(entries) == 0 {
		_ = os.Remove(s.path)
		return os.Rename(tmp, s.path)
	}
	return os.Rename(tmp, s.path)
}

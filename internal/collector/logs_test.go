package collector

import "testing"

func TestParseDockerLogLine(t *testing.T) {
	got := parseDockerLogLine("2024-06-01T12:34:56.789012345Z n8n ready on :5678")
	if got.loggedAt != "2024-06-01T12:34:56.789012345Z" {
		t.Fatalf("loggedAt = %q", got.loggedAt)
	}
	if got.message != "n8n ready on :5678" {
		t.Fatalf("message = %q", got.message)
	}
	if got.stream != "stdout" {
		t.Fatalf("stream = %q", got.stream)
	}
}

func TestParseDockerLogLineStderr(t *testing.T) {
	got := parseDockerLogLine("stderr 2024-06-01T12:34:56.789012345Z boom")
	if got.stream != "stderr" || got.message != "boom" {
		t.Fatalf("parsed = %#v", got)
	}
}

func TestLoadLogTargetsMissingFile(t *testing.T) {
	if got := loadLogTargets(t.TempDir()); len(got) != 0 {
		t.Fatalf("expected no targets, got %#v", got)
	}
}

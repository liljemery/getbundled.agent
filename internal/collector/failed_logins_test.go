package collector

import "testing"

func TestMatchesFailedSSHLine(t *testing.T) {
	if !matchesFailedSSHLine("Failed password for root from 1.2.3.4 port 22 ssh2") {
		t.Fatal("expected match")
	}
	if !matchesFailedSSHLine("Invalid user admin from 1.2.3.4 port 22") {
		t.Fatal("expected invalid user match")
	}
	if matchesFailedSSHLine("Accepted password for root from 1.2.3.4 port 22 ssh2") {
		t.Fatal("accepted login should not match")
	}
}

func TestParseLastbLineTime(t *testing.T) {
	ts, ok := parseLastbLineTime("root ssh:notty 1.2.3.4 Sun Jun 29 03:50 - 03:51 (00:01)")
	if !ok {
		t.Fatal("expected parsed time")
	}
	if ts.Hour() != 3 || ts.Minute() != 50 {
		t.Fatalf("unexpected time: %v", ts)
	}
}

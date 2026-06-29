package collector

import (
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"
)

const failedLoginWindow = 5 * time.Minute

// lastbTimeRe matches "Jun 29 03:50" inside a lastb line.
var lastbTimeRe = regexp.MustCompile(`([A-Z][a-z]{2} +\d+ \d{2}:\d{2})`)

func collectFailedLogins5Min() *int {
	if runtime.GOOS == "windows" {
		return nil
	}
	if n := countJournalFailedSSH(failedLoginWindow); n != nil {
		return n
	}
	if n := countLastbFailedSSH(failedLoginWindow); n != nil {
		return n
	}
	zero := 0
	return &zero
}

func countJournalFailedSSH(window time.Duration) *int {
	if _, err := exec.LookPath("journalctl"); err != nil {
		return nil
	}
	minutes := int(window.Minutes())
	if minutes < 1 {
		minutes = 1
	}
	out, err := exec.Command(
		"journalctl",
		"-u", "ssh.service",
		"-u", "sshd.service",
		"--since", fmt.Sprintf("%d min ago", minutes),
		"--no-pager",
		"-o", "cat",
	).CombinedOutput()
	if err != nil && len(out) == 0 {
		return nil
	}
	count := 0
	for _, line := range strings.Split(string(out), "\n") {
		if matchesFailedSSHLine(strings.TrimSpace(line)) {
			count++
		}
	}
	return &count
}

func matchesFailedSSHLine(line string) bool {
	if line == "" {
		return false
	}
	lower := strings.ToLower(line)
	return strings.Contains(lower, "failed password") ||
		strings.Contains(lower, "invalid user") ||
		strings.Contains(lower, "authentication failure")
}

func countLastbFailedSSH(window time.Duration) *int {
	if _, err := exec.LookPath("lastb"); err != nil {
		return nil
	}
	since := time.Now().Add(-window).Format("2006-01-02 15:04:05")
	out, err := exec.Command("lastb", "-s", since).CombinedOutput()
	if err != nil && len(out) == 0 {
		return nil
	}
	cutoff := time.Now().Add(-window)
	count := 0
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "btmp begins") {
			continue
		}
		if ts, ok := parseLastbLineTime(line); ok && ts.Before(cutoff) {
			continue
		}
		count++
	}
	return &count
}

func parseLastbLineTime(line string) (time.Time, bool) {
	match := lastbTimeRe.FindStringSubmatch(line)
	if len(match) < 2 {
		return time.Time{}, false
	}
	ts, err := time.ParseInLocation("Jan _2 15:04", strings.TrimSpace(match[1]), time.Local)
	if err != nil {
		return time.Time{}, false
	}
	now := time.Now()
	ts = time.Date(now.Year(), ts.Month(), ts.Day(), ts.Hour(), ts.Minute(), 0, 0, time.Local)
	if ts.After(now.Add(2 * time.Minute)) {
		ts = ts.AddDate(-1, 0, 0)
	}
	return ts, true
}

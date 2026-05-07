package compass

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LogEntry represents a parsed line from log.md.
type LogEntry struct {
	Timestamp   time.Time `json:"timestamp"`
	EventType   string    `json:"event_type"`
	Description string    `json:"description"`
	Raw         string    `json:"raw"`
}

// AppendLogEntry appends a single entry to log.md in append mode.
// Format: ## [YYYY-MM-DD HH:MM] event-type | description
func AppendLogEntry(compassPath, eventType, description string) error {
	logPath := filepath.Join(compassPath, "log.md")
	now := time.Now().Local()
	entry := fmt.Sprintf("\n## [%s] %s | %s\n",
		now.Format("2006-01-02 15:04"),
		eventType,
		description,
	)
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("opening log.md: %w", err)
	}
	defer f.Close()
	_, err = f.WriteString(entry)
	return err
}

// ParseRecentLogEntries reads log.md and returns entries from the last `days` days.
// Returns at most 100 entries even if more match. Returns empty slice if log doesn't exist.
func ParseRecentLogEntries(compassPath string, days int) ([]LogEntry, error) {
	logPath := filepath.Join(compassPath, "log.md")
	data, err := os.ReadFile(logPath)
	if os.IsNotExist(err) {
		return []LogEntry{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading log.md: %w", err)
	}

	cutoff := time.Now().AddDate(0, 0, -days)
	var entries []LogEntry

	for _, line := range strings.Split(string(data), "\n") {
		entry, ok := parseLogLine(line)
		if !ok {
			continue
		}
		if entry.Timestamp.After(cutoff) {
			entries = append(entries, entry)
		}
		if len(entries) >= 100 {
			break
		}
	}
	return entries, nil
}

// parseLogLine attempts to parse a line in format:
// ## [YYYY-MM-DD HH:MM] event-type | description
func parseLogLine(line string) (LogEntry, bool) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "## [") {
		return LogEntry{}, false
	}
	// Strip leading "## "
	rest := line[3:]
	// Find closing ]
	closeBracket := strings.Index(rest, "]")
	if closeBracket < 0 {
		return LogEntry{}, false
	}
	timeStr := rest[1:closeBracket] // inside [ ]
	rest = strings.TrimSpace(rest[closeBracket+1:])

	t, err := time.ParseInLocation("2006-01-02 15:04", timeStr, time.Local)
	if err != nil {
		return LogEntry{}, false
	}

	// Split on " | "
	parts := strings.SplitN(rest, " | ", 2)
	eventType := strings.TrimSpace(parts[0])
	description := ""
	if len(parts) == 2 {
		description = strings.TrimSpace(parts[1])
	}

	return LogEntry{
		Timestamp:   t,
		EventType:   eventType,
		Description: description,
		Raw:         line,
	}, true
}

// LastReviewDate returns the timestamp of the most recent accept or reject event in log.md.
func LastReviewDate(compassPath string) (time.Time, error) {
	entries, err := ParseRecentLogEntries(compassPath, 365)
	if err != nil {
		return time.Time{}, err
	}
	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		if e.EventType == "refinement-accepted" || e.EventType == "refinement-rejected" {
			return e.Timestamp, nil
		}
	}
	return time.Time{}, nil
}

// CountReviewEventsInDays counts accept+reject events in the last `days` days.
func CountReviewEventsInDays(compassPath string, days int) (int, error) {
	entries, err := ParseRecentLogEntries(compassPath, days)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, e := range entries {
		if e.EventType == "refinement-accepted" || e.EventType == "refinement-rejected" {
			count++
		}
	}
	return count, nil
}

// CountEventTypeInMonth counts events of a given type within the current calendar month.
func CountEventTypeInMonth(compassPath, eventType string) (int, error) {
	entries, err := ParseRecentLogEntries(compassPath, 31)
	if err != nil {
		return 0, err
	}
	now := time.Now()
	count := 0
	for _, e := range entries {
		if e.EventType == eventType && e.Timestamp.Month() == now.Month() && e.Timestamp.Year() == now.Year() {
			count++
		}
	}
	return count, nil
}

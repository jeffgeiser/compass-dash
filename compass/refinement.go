package compass

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
)

// Refinement represents a parsed refinement file.
type Refinement struct {
	ID                  string            `json:"id"`
	Filename            string            `json:"filename"`
	Frontmatter         map[string]string `json:"frontmatter"`
	ProposedAt          time.Time         `json:"proposed_at"`
	ProposedBy          string            `json:"proposed_by"`
	TargetFile          string            `json:"target_file"`
	TargetSection       string            `json:"target_section"`
	ChangeType          string            `json:"change_type"`
	Confidence          string            `json:"confidence"`
	ObservationPreview  string            `json:"observation_preview"`
	Observation         string            `json:"observation"`
	ProposedChange      string            `json:"proposed_change"`
	Reasoning           string            `json:"reasoning"`
	Evidence            string            `json:"evidence"`
	SuggestedFollowUp   string            `json:"suggested_follow_up,omitempty"`
	RawContent          string            `json:"-"`
}

// ListPendingRefinements returns all refinements in refinements/pending/, oldest first.
func ListPendingRefinements(compassPath string) ([]Refinement, error) {
	pendingDir := filepath.Join(compassPath, "refinements", "pending")
	return listRefinementsInDir(pendingDir)
}

func listRefinementsInDir(dir string) ([]Refinement, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Refinement{}, nil
		}
		return nil, fmt.Errorf("reading %s: %w", dir, err)
	}

	var refinements []Refinement
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		r, err := ParseRefinement(entry.Name(), string(content))
		if err != nil {
			continue
		}
		refinements = append(refinements, r)
	}

	sort.Slice(refinements, func(i, j int) bool {
		return refinements[i].ProposedAt.Before(refinements[j].ProposedAt)
	})
	return refinements, nil
}

// ParseRefinement parses a refinement file's content into a Refinement.
func ParseRefinement(filename, content string) (Refinement, error) {
	r := Refinement{
		Filename:    filename,
		ID:          strings.TrimSuffix(filename, ".md"),
		RawContent:  content,
		Frontmatter: make(map[string]string),
	}

	fm, body, err := splitFrontmatter(content)
	if err != nil {
		return r, fmt.Errorf("no frontmatter in %s: %w", filename, err)
	}

	r.Frontmatter = fm
	r.ProposedBy = fm["proposed_by"]
	r.TargetFile = fm["target_file"]
	r.TargetSection = strings.Trim(fm["target_section"], `"`)
	r.ChangeType = fm["change_type"]
	r.Confidence = fm["confidence"]

	if t, err := time.Parse(time.RFC3339, fm["proposed_at"]); err == nil {
		r.ProposedAt = t
	}

	r.Observation = extractSection(body, "Observation")
	r.ProposedChange = extractSection(body, "Proposed change")
	r.Reasoning = extractSection(body, "Reasoning")
	r.Evidence = extractSection(body, "Evidence")
	r.SuggestedFollowUp = extractSection(body, "Suggested follow-up")

	preview := strings.TrimSpace(r.Observation)
	if len(preview) > 200 {
		preview = preview[:197] + "..."
	}
	r.ObservationPreview = preview

	return r, nil
}

// splitFrontmatter splits a file into YAML frontmatter and body.
// Returns error if no valid frontmatter found.
func splitFrontmatter(content string) (map[string]string, string, error) {
	if !strings.HasPrefix(content, "---") {
		return nil, content, fmt.Errorf("no frontmatter delimiter")
	}
	// Find closing ---
	rest := content[3:]
	// Skip optional newline after opening ---
	if len(rest) > 0 && rest[0] == '\n' {
		rest = rest[1:]
	}
	idx := strings.Index(rest, "\n---")
	if idx == -1 {
		return nil, content, fmt.Errorf("no closing frontmatter delimiter")
	}
	fmText := rest[:idx]
	body := rest[idx+4:] // skip \n---
	body = strings.TrimLeft(body, "\n")

	fm := parseSimpleYAML(fmText)
	return fm, body, nil
}

// parseSimpleYAML parses simple key: value YAML (no nesting, no lists).
func parseSimpleYAML(text string) map[string]string {
	result := make(map[string]string)
	for _, line := range strings.Split(text, "\n") {
		// Strip inline comments
		if ci := strings.Index(line, " #"); ci >= 0 {
			line = line[:ci]
		}
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		colon := strings.Index(line, ":")
		if colon < 0 {
			continue
		}
		key := strings.TrimSpace(line[:colon])
		val := strings.TrimSpace(line[colon+1:])
		// Strip surrounding quotes
		if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'')) {
			val = val[1 : len(val)-1]
		}
		result[key] = val
	}
	return result
}

// extractSection extracts the content under a ## heading in a markdown body.
// Returns the content between that heading and the next ## heading (or EOF),
// with leading/trailing whitespace trimmed.
func extractSection(body, heading string) string {
	lines := strings.Split(body, "\n")
	headingLower := strings.ToLower(heading)

	inSection := false
	var sectionLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			sectionTitle := strings.TrimSpace(trimmed[3:])
			if inSection {
				// Hit the next section — done
				break
			}
			if normalizeHeading(sectionTitle) == headingLower {
				inSection = true
				continue
			}
		}
		if inSection {
			sectionLines = append(sectionLines, line)
		}
	}

	return strings.TrimFunc(strings.Join(sectionLines, "\n"), func(r rune) bool {
		return unicode.IsSpace(r)
	})
}

func normalizeHeading(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

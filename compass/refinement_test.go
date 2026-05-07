package compass

import (
	"testing"
	"time"
)

const sampleRefinement = `---
proposed_at: 2026-04-23T14:32:07Z
proposed_by: cowork
target_file: self/voice.md
target_section: "Vocabulary preferences"
change_type: addition
confidence: medium
session_context: email-drafting
---

## Observation

User asked me to draft an email and corrected my use of "circle back" to "follow up." This is the third time in the last month they've made this specific correction.

## Proposed change

Add to self/voice.md under "Vocabulary preferences > I avoid":
- "circle back" — user prefers "follow up"

## Reasoning

Three observed corrections of the same phrase suggests this is a stable preference, not a one-off. Confidence is medium because three instances is suggestive but not conclusive.

## Evidence

- 2026-04-15: Email draft to client; user changed "let's circle back next week" to "let's follow up next week"
- 2026-04-19: Internal memo; user changed "circling back on the proposal" to "following up on the proposal"
- 2026-04-23: Email draft (today); user changed "happy to circle back" to "happy to follow up"

## Suggested follow-up

User may have a broader aversion to corporate cliches.
`

func TestParseRefinement_Full(t *testing.T) {
	r, err := ParseRefinement("2026-04-23-143207_voice-tone-update.md", sampleRefinement)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if r.ID != "2026-04-23-143207_voice-tone-update" {
		t.Errorf("ID = %q, want %q", r.ID, "2026-04-23-143207_voice-tone-update")
	}
	if r.ProposedBy != "cowork" {
		t.Errorf("ProposedBy = %q", r.ProposedBy)
	}
	if r.TargetFile != "self/voice.md" {
		t.Errorf("TargetFile = %q", r.TargetFile)
	}
	if r.TargetSection != "Vocabulary preferences" {
		t.Errorf("TargetSection = %q", r.TargetSection)
	}
	if r.ChangeType != "addition" {
		t.Errorf("ChangeType = %q", r.ChangeType)
	}
	if r.Confidence != "medium" {
		t.Errorf("Confidence = %q", r.Confidence)
	}
	wantTime := time.Date(2026, 4, 23, 14, 32, 7, 0, time.UTC)
	if !r.ProposedAt.Equal(wantTime) {
		t.Errorf("ProposedAt = %v, want %v", r.ProposedAt, wantTime)
	}

	if r.Observation == "" {
		t.Error("Observation is empty")
	}
	if r.ProposedChange == "" {
		t.Error("ProposedChange is empty")
	}
	if r.Reasoning == "" {
		t.Error("Reasoning is empty")
	}
	if r.Evidence == "" {
		t.Error("Evidence is empty")
	}
	if r.SuggestedFollowUp == "" {
		t.Error("SuggestedFollowUp is empty")
	}

	if len(r.ObservationPreview) > 200 {
		t.Errorf("ObservationPreview too long: %d chars", len(r.ObservationPreview))
	}
}

func TestParseRefinement_NoFrontmatter(t *testing.T) {
	_, err := ParseRefinement("test.md", "## Observation\n\nsome content\n")
	if err == nil {
		t.Fatal("expected error for missing frontmatter, got nil")
	}
}

func TestParseRefinement_MinimalFrontmatter(t *testing.T) {
	content := `---
proposed_at: 2026-01-01T00:00:00Z
proposed_by: test
target_file: self/voice.md
change_type: addition
confidence: low
---

## Observation

Minimal test.

## Proposed change

Add something.

## Reasoning

Just testing.
`
	r, err := ParseRefinement("minimal.md", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.TargetSection != "" {
		t.Errorf("expected empty TargetSection, got %q", r.TargetSection)
	}
	if r.Observation != "Minimal test." {
		t.Errorf("Observation = %q", r.Observation)
	}
}

func TestSplitFrontmatter_Valid(t *testing.T) {
	content := "---\nkey: value\n---\n\nBody here.\n"
	fm, body, err := splitFrontmatter(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fm["key"] != "value" {
		t.Errorf("fm[key] = %q", fm["key"])
	}
	if body != "Body here.\n" {
		t.Errorf("body = %q", body)
	}
}

func TestSplitFrontmatter_NoDelimiter(t *testing.T) {
	_, _, err := splitFrontmatter("just content, no frontmatter")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSplitFrontmatter_UnclosedDelimiter(t *testing.T) {
	_, _, err := splitFrontmatter("---\nkey: value\n")
	if err == nil {
		t.Fatal("expected error for unclosed frontmatter, got nil")
	}
}

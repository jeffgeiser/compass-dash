package compass

import (
	"errors"
	"strings"
	"testing"
)

// ---- Addition tests --------------------------------------------------------

func TestApplyAddition_NoSection(t *testing.T) {
	content := "# Voice\n\nSome existing content.\n"
	proposed := "- new bullet item"

	result, err := ApplyChange(content, "addition", "", proposed)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.NewContent, proposed) {
		t.Errorf("proposed content not found in result:\n%s", result.NewContent)
	}
}

func TestApplyAddition_ExistingSection(t *testing.T) {
	content := `# Voice

## Vocabulary preferences

**I use:**
- direct language

**I avoid:**
- corporate speak

## Sentence structure

Some content here.
`
	proposed := `- "circle back" — use "follow up" instead`
	result, err := ApplyChange(content, "addition", "Vocabulary preferences", proposed)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.NewContent, proposed) {
		t.Errorf("proposed content not found in result:\n%s", result.NewContent)
	}
	// Must still contain the next section
	if !strings.Contains(result.NewContent, "## Sentence structure") {
		t.Errorf("next section was lost:\n%s", result.NewContent)
	}
}

func TestApplyAddition_SectionNotFound(t *testing.T) {
	content := "# Voice\n\n## Tone\n\nSome content.\n"
	_, err := ApplyChange(content, "addition", "Nonexistent section", "new item")
	if err == nil {
		t.Fatal("expected error for missing section, got nil")
	}
	if !errors.Is(err, ErrAmbiguous) {
		t.Errorf("expected ErrAmbiguous, got: %v", err)
	}
}

func TestApplyAddition_SubsectionGreaterThan(t *testing.T) {
	content := `# Voice

## Vocabulary preferences

**I use:**
- plain words

**I avoid:**
- corporate speak

## Next section
`
	proposed := `- "learnings" — use "lessons" instead`
	result, err := ApplyChange(content, "addition", "Vocabulary preferences > I avoid", proposed)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.NewContent, proposed) {
		t.Errorf("proposed content not found in result:\n%s", result.NewContent)
	}
}

func TestApplyAddition_SubsectionNotFound(t *testing.T) {
	content := "# Voice\n\n## Vocabulary preferences\n\n**I use:**\n- stuff\n"
	_, err := ApplyChange(content, "addition", "Vocabulary preferences > I avoid", "new item")
	if err == nil {
		t.Fatal("expected ErrAmbiguous for missing subsection, got nil")
	}
	if !errors.Is(err, ErrAmbiguous) {
		t.Errorf("expected ErrAmbiguous, got: %v", err)
	}
}

func TestApplyAddition_EmptyProposedChange(t *testing.T) {
	content := "# Voice\n\n## Tone\n\nContent.\n"
	_, err := ApplyChange(content, "addition", "Tone", "")
	if err == nil {
		t.Fatal("expected error for empty proposed change, got nil")
	}
	if !errors.Is(err, ErrAmbiguous) {
		t.Errorf("expected ErrAmbiguous, got: %v", err)
	}
}

// ---- Modification tests ----------------------------------------------------

func TestApplyModification_BoldBeforeAfter(t *testing.T) {
	content := "# Voice\n\n\"It's worth noting that we should perhaps consider\" a different approach.\n"
	proposedChange := `**Before (agent draft):** "It's worth noting that we should perhaps consider"

**After (my version):** "We should"`

	result, err := ApplyChange(content, "modification", "", proposedChange)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.NewContent, `"We should"`) {
		t.Errorf("after content not applied:\n%s", result.NewContent)
	}
}

func TestApplyModification_ExactMatch(t *testing.T) {
	content := "# Preferences\n\nI prefer email over Slack.\n"
	proposedChange := "**Before:** I prefer email over Slack.\n**After:** I prefer async docs over email or Slack."

	result, err := ApplyChange(content, "modification", "", proposedChange)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.NewContent, "async docs") {
		t.Errorf("modification not applied:\n%s", result.NewContent)
	}
}

func TestApplyModification_NotFound(t *testing.T) {
	content := "# Voice\n\nSome content.\n"
	proposedChange := "**Before:** text that does not exist in file\n**After:** replacement"

	_, err := ApplyChange(content, "modification", "", proposedChange)
	if err == nil {
		t.Fatal("expected ErrAmbiguous for not-found before text, got nil")
	}
	if !errors.Is(err, ErrAmbiguous) {
		t.Errorf("expected ErrAmbiguous, got: %v", err)
	}
}

func TestApplyModification_MultipleMatches(t *testing.T) {
	content := "duplicate\nduplicate\n"
	proposedChange := "**Before:** duplicate\n**After:** unique"

	_, err := ApplyChange(content, "modification", "", proposedChange)
	if err == nil {
		t.Fatal("expected ErrAmbiguous for multiple matches, got nil")
	}
	if !errors.Is(err, ErrAmbiguous) {
		t.Errorf("expected ErrAmbiguous, got: %v", err)
	}
}

func TestApplyModification_EmptyBefore(t *testing.T) {
	content := "# Voice\n\nContent.\n"
	proposedChange := "**Before:** \n**After:** something"

	_, err := ApplyChange(content, "modification", "", proposedChange)
	if err == nil {
		t.Fatal("expected ErrAmbiguous for empty before, got nil")
	}
}

func TestApplyModification_NoParsableBefore(t *testing.T) {
	content := "# Voice\n\nContent.\n"
	proposedChange := "This is just a description with no before/after markers."

	_, err := ApplyChange(content, "modification", "", proposedChange)
	if err == nil {
		t.Fatal("expected error when no before/after can be parsed, got nil")
	}
	if !errors.Is(err, ErrAmbiguous) {
		t.Errorf("expected ErrAmbiguous, got: %v", err)
	}
}

// ---- Removal tests ---------------------------------------------------------

func TestApplyRemoval_FencedBlock(t *testing.T) {
	content := "# Voice\n\n- keep this\n- remove this item\n- keep this too\n"
	proposedChange := "```\n- remove this item\n```\n"

	result, err := ApplyChange(content, "removal", "", proposedChange)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result.NewContent, "remove this item") {
		t.Errorf("target was not removed:\n%s", result.NewContent)
	}
	if !strings.Contains(result.NewContent, "keep this") {
		t.Errorf("other content was removed:\n%s", result.NewContent)
	}
}

func TestApplyRemoval_NotFound(t *testing.T) {
	content := "# Voice\n\nContent.\n"
	proposedChange := "```\ntext that is not in file\n```"

	_, err := ApplyChange(content, "removal", "", proposedChange)
	if err == nil {
		t.Fatal("expected ErrAmbiguous for not-found removal target, got nil")
	}
	if !errors.Is(err, ErrAmbiguous) {
		t.Errorf("expected ErrAmbiguous, got: %v", err)
	}
}

func TestApplyRemoval_MultipleMatches(t *testing.T) {
	content := "abc\nabc\n"
	proposedChange := "```\nabc\n```"

	_, err := ApplyChange(content, "removal", "", proposedChange)
	if err == nil {
		t.Fatal("expected ErrAmbiguous for multiple matches, got nil")
	}
}

// ---- Unknown change type ---------------------------------------------------

func TestApplyChange_UnknownType(t *testing.T) {
	_, err := ApplyChange("content", "upsert", "", "stuff")
	if err == nil {
		t.Fatal("expected error for unknown change type, got nil")
	}
}

// ---- Preview ---------------------------------------------------------------

func TestPreviewChange_ReturnsNewContent(t *testing.T) {
	content := "# Voice\n\n## Tone\n\nExisting.\n"
	preview, err := PreviewChange(content, "addition", "Tone", "New addition.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(preview, "New addition.") {
		t.Errorf("preview missing expected content:\n%s", preview)
	}
	// Original should be unchanged (preview is a pure function)
	if strings.Contains(content, "New addition.") {
		t.Errorf("original content was mutated")
	}
}

// ---- Frontmatter parsing ---------------------------------------------------

func TestParseSimpleYAML_Basic(t *testing.T) {
	text := `proposed_at: 2026-04-23T14:32:07Z
proposed_by: cowork
target_file: self/voice.md
target_section: "Vocabulary preferences"
change_type: addition
confidence: medium`

	fm := parseSimpleYAML(text)
	cases := map[string]string{
		"proposed_at":    "2026-04-23T14:32:07Z",
		"proposed_by":    "cowork",
		"target_file":    "self/voice.md",
		"target_section": "Vocabulary preferences",
		"change_type":    "addition",
		"confidence":     "medium",
	}
	for k, want := range cases {
		if got := fm[k]; got != want {
			t.Errorf("fm[%q] = %q, want %q", k, got, want)
		}
	}
}

func TestParseSimpleYAML_InlineComments(t *testing.T) {
	text := `proposed_by: cowork  # the agent
change_type: addition  # addition | modification | removal`
	fm := parseSimpleYAML(text)
	if fm["proposed_by"] != "cowork" {
		t.Errorf("inline comment not stripped, got %q", fm["proposed_by"])
	}
	if fm["change_type"] != "addition" {
		t.Errorf("inline comment not stripped, got %q", fm["change_type"])
	}
}

// ---- Section extraction ----------------------------------------------------

func TestExtractSection_Basic(t *testing.T) {
	body := `## Observation

The agent observed that the user corrected "circle back" to "follow up" three times.

## Proposed change

Add to voice.md.

## Reasoning

High confidence.
`
	obs := extractSection(body, "Observation")
	if !strings.Contains(obs, "circle back") {
		t.Errorf("observation section not extracted correctly, got: %q", obs)
	}

	proposed := extractSection(body, "Proposed change")
	if !strings.Contains(proposed, "Add to voice.md") {
		t.Errorf("proposed change section not extracted correctly, got: %q", proposed)
	}
}

func TestExtractSection_CaseInsensitive(t *testing.T) {
	body := "## OBSERVATION\n\nsome content\n\n## Next\n"
	obs := extractSection(body, "observation")
	if obs != "some content" {
		t.Errorf("expected 'some content', got: %q", obs)
	}
}

func TestExtractSection_Missing(t *testing.T) {
	body := "## Observation\n\ncontent\n"
	result := extractSection(body, "Nonexistent")
	if result != "" {
		t.Errorf("expected empty string for missing section, got: %q", result)
	}
}

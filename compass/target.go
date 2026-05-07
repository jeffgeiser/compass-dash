// Package compass/target.go — applies refinement changes to target files.
//
// This is the most critical file in the codebase. It is the only place where
// user Compass content is modified automatically. The implementation is
// deliberately conservative: if a change is ambiguous or the target cannot be
// found exactly, it returns an error and does NOT modify the file. The caller
// should surface the error and prompt the user to apply the change manually
// via the Edit & Accept flow.
package compass

import (
	"errors"
	"fmt"
	"strings"
)

// ErrAmbiguous is returned when a change cannot be applied unambiguously.
var ErrAmbiguous = errors.New("change is ambiguous and cannot be applied automatically")

// ApplyResult holds the result of applying a refinement to file content.
type ApplyResult struct {
	NewContent string
	Applied    bool
	Message    string // human-readable explanation of what happened
}

// ApplyChange applies a proposed change to existing file content.
// It inspects the changeType and dispatches to the appropriate method.
// Returns an error (wrapping ErrAmbiguous where appropriate) if the change
// cannot be applied safely.
func ApplyChange(currentContent, changeType, targetSection, proposedChange string) (ApplyResult, error) {
	switch strings.ToLower(changeType) {
	case "addition":
		return applyAddition(currentContent, targetSection, proposedChange)
	case "modification":
		return applyModification(currentContent, proposedChange)
	case "removal":
		return applyRemoval(currentContent, proposedChange)
	default:
		return ApplyResult{}, fmt.Errorf("unknown change_type %q: expected addition, modification, or removal", changeType)
	}
}

// applyAddition appends proposedChange to the end of the named section.
// If targetSection is empty, appends to the end of the file.
// If targetSection is specified but not found, returns ErrAmbiguous.
func applyAddition(content, targetSection, proposedChange string) (ApplyResult, error) {
	proposed := strings.TrimSpace(proposedChange)
	if proposed == "" {
		return ApplyResult{}, fmt.Errorf("%w: proposed change is empty", ErrAmbiguous)
	}

	if targetSection == "" {
		// No section specified — append to end of file
		sep := "\n"
		if !strings.HasSuffix(content, "\n") {
			sep = "\n\n"
		} else if !strings.HasSuffix(content, "\n\n") {
			sep = "\n"
		}
		newContent := content + sep + proposed + "\n"
		return ApplyResult{
			NewContent: newContent,
			Applied:    true,
			Message:    "Appended to end of file",
		}, nil
	}

	// Find the target section and append the content inside it.
	lines := strings.Split(content, "\n")
	sectionLine := -1
	sectionEnd := len(lines)
	targetNorm := normalizeHeading(targetSection)

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			title := normalizeHeading(trimmed[3:])
			if title == targetNorm {
				sectionLine = i
				continue
			}
			if sectionLine >= 0 {
				sectionEnd = i
				break
			}
		}
		// Also handle ### subsections if target_section uses > syntax
	}

	if sectionLine < 0 {
		// Section not found — try subsection matching if target has > syntax
		return trySubsectionAddition(content, targetSection, proposed)
	}

	// Find the last non-empty line in the section (before sectionEnd)
	insertAt := sectionEnd
	for i := sectionEnd - 1; i > sectionLine; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			insertAt = i + 1
			break
		}
	}

	// Build new content
	var sb strings.Builder
	for i, line := range lines {
		if i == insertAt {
			sb.WriteString("\n")
			sb.WriteString(proposed)
			sb.WriteString("\n")
		}
		sb.WriteString(line)
		if i < len(lines)-1 {
			sb.WriteString("\n")
		}
	}
	if insertAt >= len(lines) {
		sb.WriteString("\n")
		sb.WriteString(proposed)
		sb.WriteString("\n")
	}

	return ApplyResult{
		NewContent: sb.String(),
		Applied:    true,
		Message:    fmt.Sprintf("Appended to section %q", targetSection),
	}, nil
}

// trySubsectionAddition handles target_section values like "Vocabulary preferences > I avoid"
// by looking for the subsection (### heading or list block) within the parent section.
func trySubsectionAddition(content, targetSection, proposed string) (ApplyResult, error) {
	// Split on " > " to get parent and child
	parts := strings.SplitN(targetSection, " > ", 2)
	if len(parts) != 2 {
		return ApplyResult{}, fmt.Errorf("%w: section %q not found in target file", ErrAmbiguous, targetSection)
	}
	parentSection := strings.TrimSpace(parts[0])
	childHeading := strings.TrimSpace(parts[1])

	lines := strings.Split(content, "\n")
	parentNorm := normalizeHeading(parentSection)
	childNorm := normalizeHeading(childHeading)

	inParent := false
	childLine := -1
	childEnd := len(lines)

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			title := normalizeHeading(trimmed[3:])
			if title == parentNorm {
				inParent = true
				continue
			}
			if inParent {
				// Leaving the parent section — record end if we already found child
				if childLine >= 0 {
					childEnd = i
				}
				break
			}
		}
		if inParent && strings.HasPrefix(trimmed, "### ") {
			title := normalizeHeading(trimmed[4:])
			if title == childNorm {
				childLine = i
				continue
			}
			if childLine >= 0 {
				childEnd = i
				break
			}
		}
		// Handle **Label:** style bold sub-headings like "**I avoid:** content"
		if inParent && strings.HasPrefix(trimmed, "**") {
			rest := trimmed[2:] // strip opening **
			if closeIdx := strings.Index(rest, "**"); closeIdx >= 0 {
				label := strings.TrimRight(rest[:closeIdx], ":")
				if childLine >= 0 {
					// A new bold subsection ends the previous one
					childEnd = i
					break
				}
				if normalizeHeading(strings.TrimSpace(label)) == childNorm {
					childLine = i
				}
			}
		}
	}

	if childLine < 0 {
		return ApplyResult{}, fmt.Errorf("%w: section %q not found in target file (parent %q found, child %q not found)", ErrAmbiguous, targetSection, parentSection, childHeading)
	}

	// Insert after the last non-empty line in the child section
	insertAt := childEnd
	for i := childEnd - 1; i > childLine; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			insertAt = i + 1
			break
		}
	}

	var sb strings.Builder
	for i, line := range lines {
		if i == insertAt {
			sb.WriteString(proposed)
			sb.WriteString("\n")
			// Ensure blank line before a heading
			if strings.HasPrefix(strings.TrimSpace(line), "#") {
				sb.WriteString("\n")
			}
		}
		sb.WriteString(line)
		if i < len(lines)-1 {
			sb.WriteString("\n")
		}
	}
	if insertAt >= len(lines) {
		sb.WriteString(proposed)
		sb.WriteString("\n")
	}

	return ApplyResult{
		NewContent: sb.String(),
		Applied:    true,
		Message:    fmt.Sprintf("Appended to subsection %q", targetSection),
	}, nil
}

// applyModification replaces "before" text with "after" text.
// The proposedChange must contain clear before/after delimiters.
// If the before text is not found exactly in content, returns ErrAmbiguous.
func applyModification(content, proposedChange string) (ApplyResult, error) {
	before, after, err := parseBeforeAfter(proposedChange)
	if err != nil {
		return ApplyResult{}, fmt.Errorf("%w: %v", ErrAmbiguous, err)
	}

	if before == "" {
		return ApplyResult{}, fmt.Errorf("%w: modification has empty 'before' text", ErrAmbiguous)
	}

	count := strings.Count(content, before)
	if count == 0 {
		return ApplyResult{}, fmt.Errorf("%w: 'before' text not found exactly in target file — use Edit & Accept to apply manually", ErrAmbiguous)
	}
	if count > 1 {
		return ApplyResult{}, fmt.Errorf("%w: 'before' text appears %d times in target file — use Edit & Accept to apply manually", ErrAmbiguous, count)
	}

	newContent := strings.Replace(content, before, after, 1)
	return ApplyResult{
		NewContent: newContent,
		Applied:    true,
		Message:    "Applied modification",
	}, nil
}

// parseBeforeAfter extracts before/after blocks from a proposed change.
// Supports these formats:
//
//	**Before:** / **After:**
//	Before: / After:  (with following content)
//	```before ... ``` / ```after ... ```
func parseBeforeAfter(proposedChange string) (before, after string, err error) {
	// Try **Before (agent draft):** / **After (my version):** style
	before, after, ok := extractLabeledBlocks(proposedChange, "before", "after")
	if ok && (before != "" || after != "") {
		return before, after, nil
	}

	// Try fenced code blocks labeled before/after
	before, after, ok = extractFencedBeforeAfter(proposedChange)
	if ok {
		return before, after, nil
	}

	return "", "", fmt.Errorf("could not identify before/after blocks in proposed change; use Edit & Accept")
}

// extractLabeledBlocks looks for **Before...**: and **After...**:  style markers.
func extractLabeledBlocks(text, beforeLabel, afterLabel string) (before, after string, ok bool) {
	lines := strings.Split(text, "\n")
	state := 0 // 0=searching, 1=in-before, 2=in-after
	var beforeLines, afterLines []string

	for _, line := range lines {
		lower := strings.ToLower(strings.TrimSpace(line))
		isBeforeHeader := strings.Contains(lower, beforeLabel+":") || strings.Contains(lower, "**"+beforeLabel)
		isAfterHeader := strings.Contains(lower, afterLabel+":") || strings.Contains(lower, "**"+afterLabel)

		if isBeforeHeader && !strings.Contains(lower, afterLabel) {
			state = 1
			// Capture inline content after the colon if any.
			// Strip leading * characters from markdown bold markers like **Before:**
			if ci := strings.Index(line, ":"); ci >= 0 {
				inline := strings.TrimLeft(strings.TrimSpace(line[ci+1:]), "*")
				inline = strings.TrimSpace(inline)
				if inline != "" {
					beforeLines = append(beforeLines, inline)
				}
			}
			ok = true
			continue
		}
		if isAfterHeader {
			state = 2
			if ci := strings.Index(line, ":"); ci >= 0 {
				inline := strings.TrimLeft(strings.TrimSpace(line[ci+1:]), "*")
				inline = strings.TrimSpace(inline)
				if inline != "" {
					afterLines = append(afterLines, inline)
				}
			}
			ok = true
			continue
		}
		switch state {
		case 1:
			beforeLines = append(beforeLines, line)
		case 2:
			afterLines = append(afterLines, line)
		}
	}

	before = strings.TrimSpace(strings.Join(beforeLines, "\n"))
	after = strings.TrimSpace(strings.Join(afterLines, "\n"))
	return before, after, ok
}

// extractFencedBeforeAfter looks for ```before ... ``` / ```after ... ``` blocks.
func extractFencedBeforeAfter(text string) (before, after string, ok bool) {
	lines := strings.Split(text, "\n")
	state := 0 // 0=searching, 1=in-before-fence, 2=in-after-fence
	var beforeLines, afterLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			lang := strings.ToLower(strings.TrimSpace(trimmed[3:]))
			switch state {
			case 0:
				if lang == "before" || lang == "diff-before" {
					state = 1
					ok = true
					continue
				}
				if lang == "after" || lang == "diff-after" {
					state = 2
					ok = true
					continue
				}
			case 1:
				if trimmed == "```" {
					state = 0
					continue
				}
			case 2:
				if trimmed == "```" {
					state = 0
					continue
				}
			}
		}
		switch state {
		case 1:
			beforeLines = append(beforeLines, line)
		case 2:
			afterLines = append(afterLines, line)
		}
	}

	before = strings.TrimSpace(strings.Join(beforeLines, "\n"))
	after = strings.TrimSpace(strings.Join(afterLines, "\n"))
	return before, after, ok
}

// applyRemoval removes the specified content from the file.
// The proposedChange should contain the exact text to remove.
// If not found exactly once, returns ErrAmbiguous.
func applyRemoval(content, proposedChange string) (ApplyResult, error) {
	// Extract the content to remove — look for a block or use the whole proposedChange
	toRemove := extractRemovalTarget(proposedChange)
	if toRemove == "" {
		return ApplyResult{}, fmt.Errorf("%w: could not identify what to remove from proposed change — use Edit & Accept", ErrAmbiguous)
	}

	count := strings.Count(content, toRemove)
	if count == 0 {
		return ApplyResult{}, fmt.Errorf("%w: removal target not found in target file — use Edit & Accept", ErrAmbiguous)
	}
	if count > 1 {
		return ApplyResult{}, fmt.Errorf("%w: removal target appears %d times in target file — use Edit & Accept to apply manually", ErrAmbiguous, count)
	}

	newContent := strings.Replace(content, toRemove, "", 1)
	// Clean up double blank lines left by removal
	for strings.Contains(newContent, "\n\n\n") {
		newContent = strings.ReplaceAll(newContent, "\n\n\n", "\n\n")
	}

	return ApplyResult{
		NewContent: newContent,
		Applied:    true,
		Message:    "Applied removal",
	}, nil
}

// extractRemovalTarget extracts the text to remove from a proposed change description.
// Looks for fenced code blocks or quoted content; falls back to the whole proposedChange.
func extractRemovalTarget(proposedChange string) string {
	// Look for a fenced block
	lines := strings.Split(proposedChange, "\n")
	inFence := false
	var fenceLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") && !inFence {
			inFence = true
			continue
		}
		if trimmed == "```" && inFence {
			break
		}
		if inFence {
			fenceLines = append(fenceLines, line)
		}
	}
	if len(fenceLines) > 0 {
		return strings.TrimSpace(strings.Join(fenceLines, "\n"))
	}

	// Fall back: look for a line starting with "Remove:" or similar
	for _, line := range lines {
		lower := strings.ToLower(strings.TrimSpace(line))
		if strings.HasPrefix(lower, "remove:") || strings.HasPrefix(lower, "delete:") {
			candidate := strings.TrimSpace(line[strings.Index(line, ":")+1:])
			if candidate != "" {
				return candidate
			}
		}
	}

	// Last resort: return the trimmed whole proposedChange
	trimmed := strings.TrimSpace(proposedChange)
	if len(trimmed) > 0 {
		return trimmed
	}
	return ""
}

// PreviewChange returns what the file content would look like after applying
// the change, without modifying any files. Returns an error if the change
// is ambiguous (same conditions as ApplyChange).
func PreviewChange(currentContent, changeType, targetSection, proposedChange string) (string, error) {
	result, err := ApplyChange(currentContent, changeType, targetSection, proposedChange)
	if err != nil {
		return "", err
	}
	return result.NewContent, nil
}

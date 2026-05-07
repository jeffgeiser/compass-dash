package compass

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Compass holds the path to a Compass folder and provides high-level operations.
type Compass struct {
	Path string
}

// New returns a Compass for the given folder path.
func New(path string) *Compass {
	return &Compass{Path: path}
}

// ValidationError describes a single validation failure.
type ValidationError struct {
	Path    string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Path, e.Message)
}

// Validate checks that the Compass folder meets the minimum structural requirements.
func (c *Compass) Validate() []ValidationError {
	var errs []ValidationError
	check := func(rel, msg string) {
		p := filepath.Join(c.Path, rel)
		if _, err := os.Stat(p); os.IsNotExist(err) {
			errs = append(errs, ValidationError{Path: rel, Message: msg})
		}
	}
	check("COMPASS.md", "missing COMPASS.md schema file")
	check("refinements/pending", "missing refinements/pending/ directory")
	check("refinements/accepted", "missing refinements/accepted/ directory")
	check("refinements/rejected", "missing refinements/rejected/ directory")
	return errs
}

// Stats holds the Compass health summary.
type Stats struct {
	PendingCount          int        `json:"pending_count"`
	OldestPendingAgeDays  float64    `json:"oldest_pending_age_days"`
	AcceptedThisMonth     int        `json:"accepted_this_month"`
	RejectedThisMonth     int        `json:"rejected_this_month"`
	LastReviewDate        *time.Time `json:"last_review_date"`
	SubstrateFilesCount   int        `json:"substrate_files_count"`
	LintReportAvailable   bool       `json:"lint_report_available"`
}

// GetStats returns the current stats for the Compass.
func (c *Compass) GetStats() (Stats, error) {
	pending, err := ListPendingRefinements(c.Path)
	if err != nil {
		return Stats{}, err
	}

	var oldest float64
	if len(pending) > 0 {
		oldest = time.Since(pending[0].ProposedAt).Hours() / 24
	}

	accepted, _ := CountEventTypeInMonth(c.Path, "refinement-accepted")
	rejected, _ := CountEventTypeInMonth(c.Path, "refinement-rejected")

	lastReview, _ := LastReviewDate(c.Path)
	var lastReviewPtr *time.Time
	if !lastReview.IsZero() {
		lastReviewPtr = &lastReview
	}

	substrateCount := c.countSubstrateFiles()
	lintAvail := c.lintReportAvailable()

	return Stats{
		PendingCount:         len(pending),
		OldestPendingAgeDays: oldest,
		AcceptedThisMonth:    accepted,
		RejectedThisMonth:    rejected,
		LastReviewDate:       lastReviewPtr,
		SubstrateFilesCount:  substrateCount,
		LintReportAvailable:  lintAvail,
	}, nil
}

func (c *Compass) countSubstrateFiles() int {
	count := 0
	_ = filepath.Walk(c.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(c.Path, path)
		// Skip refinement files, log, COMPASS.md itself
		if strings.HasPrefix(rel, "refinements"+string(filepath.Separator)) ||
			strings.HasPrefix(rel, "lint"+string(filepath.Separator)) ||
			rel == "log.md" || rel == "COMPASS.md" || rel == "INDEX.md" {
			return nil
		}
		if strings.HasSuffix(rel, ".md") && !strings.HasPrefix(filepath.Base(rel), ".") {
			count++
		}
		return nil
	})
	return count
}

func (c *Compass) lintReportAvailable() bool {
	lintDir := filepath.Join(c.Path, "lint")
	entries, err := os.ReadDir(lintDir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			return true
		}
	}
	return false
}

// FileEntry represents a file in the Compass folder listing.
type FileEntry struct {
	Path     string `json:"path"`
	Dir      string `json:"dir"`
	Filename string `json:"filename"`
}

// ListFiles returns all markdown files in the Compass, excluding refinement and git files.
func (c *Compass) ListFiles() ([]FileEntry, error) {
	var files []FileEntry
	err := filepath.Walk(c.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(c.Path, path)
		// Skip hidden files and git
		if strings.HasPrefix(filepath.Base(rel), ".") {
			return nil
		}
		// Skip refinement history files (keep COMPASS.md, log, substrate files)
		if strings.HasPrefix(rel, "refinements"+string(filepath.Separator)+"accepted") ||
			strings.HasPrefix(rel, "refinements"+string(filepath.Separator)+"rejected") {
			return nil
		}
		if strings.HasSuffix(rel, ".md") {
			files = append(files, FileEntry{
				Path:     rel,
				Dir:      filepath.Dir(rel),
				Filename: filepath.Base(rel),
			})
		}
		return nil
	})
	return files, err
}

// ReadFile reads a file from the Compass folder by relative path.
// Returns an error if the path attempts to escape the Compass folder.
func (c *Compass) ReadFile(relPath string) (string, error) {
	base, err := filepath.Abs(c.Path)
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(filepath.Join(c.Path, filepath.Clean(relPath)))
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(abs, base+string(filepath.Separator)) && abs != base {
		return "", fmt.Errorf("path %q is outside the Compass folder", relPath)
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// MoveRefinement moves a refinement file from pending/ to the destination subfolder.
// destination should be "accepted" or "rejected".
func (c *Compass) MoveRefinement(filename, destination string) error {
	if destination != "accepted" && destination != "rejected" {
		return fmt.Errorf("invalid destination %q: must be accepted or rejected", destination)
	}
	src := filepath.Join(c.Path, "refinements", "pending", filename)
	dst := filepath.Join(c.Path, "refinements", destination, filename)

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("creating destination directory: %w", err)
	}

	// If a file with the same name exists in destination, add a suffix
	if _, err := os.Stat(dst); err == nil {
		base := strings.TrimSuffix(filename, ".md")
		dst = filepath.Join(c.Path, "refinements", destination,
			base+"-"+time.Now().Format("150405")+".md")
	}

	return os.Rename(src, dst)
}

// AppendToRefinement appends content to a refinement file in pending/.
func (c *Compass) AppendToRefinement(filename, content string) error {
	path := filepath.Join(c.Path, "refinements", "pending", filename)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening refinement file: %w", err)
	}
	defer f.Close()
	_, err = f.WriteString(content)
	return err
}

// WriteTargetFile writes content to a target file within the Compass folder.
// Refuses to write outside the Compass folder.
func (c *Compass) WriteTargetFile(relPath, content string) error {
	base, err := filepath.Abs(c.Path)
	if err != nil {
		return err
	}
	abs, err := filepath.Abs(filepath.Join(c.Path, filepath.Clean(relPath)))
	if err != nil {
		return err
	}
	if !strings.HasPrefix(abs, base+string(filepath.Separator)) {
		return fmt.Errorf("path %q is outside the Compass folder", relPath)
	}
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
		return fmt.Errorf("creating parent directory: %w", err)
	}
	return os.WriteFile(abs, []byte(content), 0644)
}

// CountInDir returns the number of .md files in a subdirectory.
func (c *Compass) CountInDir(subdir string) int {
	dir := filepath.Join(c.Path, subdir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") && !strings.HasPrefix(e.Name(), ".") {
			count++
		}
	}
	return count
}

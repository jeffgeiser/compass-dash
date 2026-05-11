package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/jeffgeiser/compass-dash/compass"
	"github.com/jeffgeiser/compass-dash/config"
)

// writeJSON encodes v as JSON and writes it with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError writes a plain-text error response.
func writeError(w http.ResponseWriter, status int, msg string) {
	http.Error(w, msg, status)
}

// decodeJSON decodes the request body into v. Returns false and writes the
// error response if decoding fails.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return false
	}
	return true
}

// getCompass returns a validated Compass for the currently configured path,
// or writes an error response and returns nil.
func getCompass(s *Server, w http.ResponseWriter) *compass.Compass {
	s.mu.RLock()
	path := s.cfg.CompassPath
	s.mu.RUnlock()

	if path == "" {
		writeError(w, http.StatusBadRequest, "compass path not configured — set it in Settings")
		return nil
	}
	c := compass.New(path)
	errs := c.Validate()
	if len(errs) > 0 {
		msgs := make([]string, len(errs))
		for i, e := range errs {
			msgs[i] = e.Error()
		}
		writeError(w, http.StatusBadRequest, "invalid compass: "+strings.Join(msgs, "; "))
		return nil
	}
	return c
}

// ── /api/health ───────────────────────────────────────────────────────────────

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"version": s.version})
}

// ── /api/config ───────────────────────────────────────────────────────────────

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	cfg := s.cfg
	s.mu.RUnlock()
	writeJSON(w, http.StatusOK, cfg)
}

func (s *Server) handlePutConfig(w http.ResponseWriter, r *http.Request) {
	var body struct {
		CompassPath string `json:"compass_path"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}

	s.mu.Lock()
	s.cfg.CompassPath = body.CompassPath
	cfg := s.cfg
	s.mu.Unlock()

	if err := config.Save(cfg); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save config: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

// ── /api/stats ────────────────────────────────────────────────────────────────

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	c := getCompass(s, w)
	if c == nil {
		return
	}
	stats, err := c.GetStats()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

// ── /api/refinements ─────────────────────────────────────────────────────────

func (s *Server) handleListRefinements(w http.ResponseWriter, r *http.Request) {
	c := getCompass(s, w)
	if c == nil {
		return
	}
	refinements, err := compass.ListPendingRefinements(c.Path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if refinements == nil {
		refinements = []compass.Refinement{}
	}
	writeJSON(w, http.StatusOK, refinements)
}

// refinementIDFromPath extracts the refinement ID (filename without .md) from
// a path like /api/refinements/{id}/action.
func refinementIDFromPath(path, action string) string {
	// path: /api/refinements/{id}/action  OR  /api/refinements/{id}
	path = strings.TrimPrefix(path, "/api/refinements/")
	if action != "" {
		path = strings.TrimSuffix(path, "/"+action)
	}
	return path
}

func (s *Server) handleRefinementPreview(w http.ResponseWriter, r *http.Request) {
	id := refinementIDFromPath(r.URL.Path, "preview")
	c := getCompass(s, w)
	if c == nil {
		return
	}

	refs, err := compass.ListPendingRefinements(c.Path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var ref *compass.Refinement
	for i := range refs {
		if refs[i].ID == id {
			ref = &refs[i]
			break
		}
	}
	if ref == nil {
		writeError(w, http.StatusNotFound, "refinement not found: "+id)
		return
	}

	if ref.TargetFile == "" {
		writeError(w, http.StatusBadRequest, "refinement has no target_file")
		return
	}

	currentContent, err := c.ReadFile(ref.TargetFile)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read target file: "+err.Error())
		return
	}

	preview, err := compass.PreviewChange(currentContent, ref.ChangeType, ref.TargetSection, ref.ProposedChange)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"preview_content": preview})
}

func (s *Server) handleRefinementAccept(w http.ResponseWriter, r *http.Request) {
	id := refinementIDFromPath(r.URL.Path, "accept")
	c := getCompass(s, w)
	if c == nil {
		return
	}

	refs, err := compass.ListPendingRefinements(c.Path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var ref *compass.Refinement
	for i := range refs {
		if refs[i].ID == id {
			ref = &refs[i]
			break
		}
	}
	if ref == nil {
		writeError(w, http.StatusNotFound, "refinement not found: "+id)
		return
	}

	if ref.TargetFile == "" {
		writeError(w, http.StatusBadRequest, "refinement has no target_file — use edit & accept")
		return
	}

	currentContent, err := c.ReadFile(ref.TargetFile)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read target file: "+err.Error())
		return
	}

	result, err := compass.ApplyChange(currentContent, ref.ChangeType, ref.TargetSection, ref.ProposedChange)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	if err := c.WriteTargetFile(ref.TargetFile, result.NewContent); err != nil {
		writeError(w, http.StatusInternalServerError, "could not write target file: "+err.Error())
		return
	}

	if err := c.MoveRefinement(ref.Filename, "accepted"); err != nil {
		writeError(w, http.StatusInternalServerError, "could not archive refinement: "+err.Error())
		return
	}

	desc := fmt.Sprintf("%s — %s", id, ref.TargetFile)
	_ = compass.AppendLogEntry(c.Path, "refinement-accepted", desc)
	gitSync(c.Path, "accepted: "+id)

	writeJSON(w, http.StatusOK, map[string]string{"status": "accepted", "message": result.Message})
}

func (s *Server) handleRefinementAcceptEdited(w http.ResponseWriter, r *http.Request) {
	id := refinementIDFromPath(r.URL.Path, "accept-edited")
	c := getCompass(s, w)
	if c == nil {
		return
	}

	refs, err := compass.ListPendingRefinements(c.Path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var ref *compass.Refinement
	for i := range refs {
		if refs[i].ID == id {
			ref = &refs[i]
			break
		}
	}
	if ref == nil {
		writeError(w, http.StatusNotFound, "refinement not found: "+id)
		return
	}

	var body struct {
		Content string `json:"content"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}

	if ref.TargetFile != "" {
		if err := c.WriteTargetFile(ref.TargetFile, body.Content); err != nil {
			writeError(w, http.StatusInternalServerError, "could not write target file: "+err.Error())
			return
		}
	}

	if err := c.MoveRefinement(ref.Filename, "accepted"); err != nil {
		writeError(w, http.StatusInternalServerError, "could not archive refinement: "+err.Error())
		return
	}

	desc := fmt.Sprintf("%s — %s (manually edited)", id, ref.TargetFile)
	_ = compass.AppendLogEntry(c.Path, "refinement-accepted", desc)
	gitSync(c.Path, "accepted: "+id)

	writeJSON(w, http.StatusOK, map[string]string{"status": "accepted"})
}

func (s *Server) handleRefinementReject(w http.ResponseWriter, r *http.Request) {
	id := refinementIDFromPath(r.URL.Path, "reject")
	c := getCompass(s, w)
	if c == nil {
		return
	}

	refs, err := compass.ListPendingRefinements(c.Path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var ref *compass.Refinement
	for i := range refs {
		if refs[i].ID == id {
			ref = &refs[i]
			break
		}
	}
	if ref == nil {
		writeError(w, http.StatusNotFound, "refinement not found: "+id)
		return
	}

	var body struct {
		Reason string `json:"reason"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}

	if body.Reason != "" {
		rejection := "\n\n## Rejection reason\n\n" + body.Reason + "\n"
		if err := c.AppendToRefinement(ref.Filename, rejection); err != nil {
			writeError(w, http.StatusInternalServerError, "could not append rejection reason: "+err.Error())
			return
		}
	}

	if err := c.MoveRefinement(ref.Filename, "rejected"); err != nil {
		writeError(w, http.StatusInternalServerError, "could not archive refinement: "+err.Error())
		return
	}

	desc := id
	if body.Reason != "" {
		desc += " | " + body.Reason
	}
	_ = compass.AppendLogEntry(c.Path, "refinement-rejected", desc)
	gitSync(c.Path, "rejected: "+id)

	writeJSON(w, http.StatusOK, map[string]string{"status": "rejected"})
}

// ── /api/files ────────────────────────────────────────────────────────────────

func (s *Server) handleListFiles(w http.ResponseWriter, r *http.Request) {
	c := getCompass(s, w)
	if c == nil {
		return
	}
	files, err := c.ListFiles()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if files == nil {
		files = []compass.FileEntry{}
	}
	writeJSON(w, http.StatusOK, files)
}

func (s *Server) handleReadFile(w http.ResponseWriter, r *http.Request) {
	relPath := strings.TrimPrefix(r.URL.Path, "/api/files/")
	if relPath == "" {
		writeError(w, http.StatusBadRequest, "file path required")
		return
	}

	c := getCompass(s, w)
	if c == nil {
		return
	}

	content, err := c.ReadFile(relPath)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"path": relPath, "content": content})
}

// ── /api/activity ─────────────────────────────────────────────────────────────

func (s *Server) handleActivity(w http.ResponseWriter, r *http.Request) {
	c := getCompass(s, w)
	if c == nil {
		return
	}

	entries, err := compass.ParseRecentLogEntries(c.Path, 30)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if entries == nil {
		entries = []compass.LogEntry{}
	}
	writeJSON(w, http.StatusOK, entries)
}

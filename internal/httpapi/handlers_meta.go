package httpapi

import (
	"net/http"
)

// handleStats 返回各实体计数概览。
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	// Stats is a read-through view over persisted entity counts.
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	stats, err := s.svc.Stats()
	if err != nil {
		writeError(w, statusForError(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

// handleSelfCheck 校验数据库不变量，返回问题列表（空表示通过）。
func (s *Server) handleSelfCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	problems, err := s.svc.SelfCheck()
	if err != nil {
		writeError(w, statusForError(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": len(problems) == 0, "problems": problems})
}

// handleHealth 健康检查端点。
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

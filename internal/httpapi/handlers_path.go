package httpapi

import (
	"encoding/json"
	"net/http"

	"task253-birdbanding/internal/model"
)

// handleEdges 处理迁徙边集合：GET 列表（按 individual_id 过滤）。
func (s *Server) handleEdges(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	q := r.URL.Query()
	edges, err := s.svc.DB().ListEdges(q.Get("individual_id"))
	if err != nil {
		writeError(w, statusForError(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, edges)
}

// handleEdgeByID 处理单个迁徙边：GET 详情 / POST 裁决（keep-rare/exclude-overspeed/confirm）。
func (s *Server) handleEdgeByID(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.URL.Path, "/api/edges/")
	if id == "" {
		writeError(w, http.StatusBadRequest, "缺少迁徙边 ID")
		return
	}
	switch r.Method {
	case http.MethodGet:
		edge, err := s.svc.DB().GetEdge(id)
		if err != nil {
			writeError(w, statusForError(err), err.Error())
			return
		}
		writeJSON(w, http.StatusOK, edge)
	case http.MethodPost:
		action := parseAction(r.URL.Path, "/api/edges/"+id+"/")
		var err error
		switch action {
		case "keep-rare":
			err = s.svc.KeepRare(id)
		case "exclude-overspeed":
			err = s.svc.ExcludeOverSpeed(id)
		case "confirm":
			err = s.svc.ConfirmEdge(id)
		default:
			writeError(w, http.StatusBadRequest, "未知动作: "+action)
			return
		}
		if err != nil {
			writeError(w, statusForError(err), err.Error())
			return
		}
		edge, _ := s.svc.DB().GetEdge(id)
		writeJSON(w, http.StatusOK, edge)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

var _ = json.Marshal
var _ = model.EdgeCandidate

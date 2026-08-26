package httpapi

import (
	"encoding/json"
	"net/http"

	"task253-birdbanding/internal/model"
)

// handleIndividuals 处理个体集合：GET 列表 / POST 关联创建。
func (s *Server) handleIndividuals(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		inds, err := s.svc.DB().ListIndividuals()
		if err != nil {
			writeError(w, statusForError(err), err.Error())
			return
		}
		writeJSON(w, http.StatusOK, inds)
	case http.MethodPost:
		var body struct {
			RingCode string `json:"ring_code"`
			Species  string `json:"species"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.RingCode == "" {
			writeError(w, http.StatusBadRequest, "ring_code 必填")
			return
		}
		ind, _, err := s.svc.ResolveIndividual(body.RingCode, body.Species)
		if err != nil {
			writeError(w, statusForError(err), err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, ind)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleIndividualByID 处理单个个体及子资源。
func (s *Server) handleIndividualByID(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.URL.Path, "/api/individuals/")
	if id == "" {
		writeError(w, http.StatusBadRequest, "缺少个体 ID")
		return
	}
	switch r.Method {
	case http.MethodGet:
		ind, err := s.svc.DB().GetIndividual(id)
		if err != nil {
			writeError(w, statusForError(err), err.Error())
			return
		}
		writeJSON(w, http.StatusOK, ind)
	case http.MethodPost:
		action := parseAction(r.URL.Path, "/api/individuals/"+id+"/")
		if action == "build-edges" {
			edges, err := s.svc.BuildEdges(id)
			if err != nil {
				writeError(w, statusForError(err), err.Error())
				return
			}
			writeJSON(w, http.StatusCreated, edges)
			return
		}
		writeError(w, http.StatusBadRequest, "未知动作: "+action)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleIndividualEvents 返回个体事件时间线（GET）。
func (s *Server) handleIndividualEvents(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.URL.Path, "/api/individuals/")
	if id == "" {
		writeError(w, http.StatusBadRequest, "缺少个体 ID")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	timeline, err := s.svc.GetIndividualTimeline(id)
	if err != nil {
		writeError(w, statusForError(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, timeline)
}

var _ = model.EventBanding

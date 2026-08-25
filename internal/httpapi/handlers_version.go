package httpapi

import (
	"encoding/json"
	"net/http"

	"task253-birdbanding/internal/model"
)

// handleVersions 处理路径版本集合：GET 列表 / POST 创建草稿。
func (s *Server) handleVersions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		q := r.URL.Query()
		versions, err := s.svc.DB().ListVersions(q.Get("individual_id"))
		if err != nil {
			writeError(w, statusForError(err), err.Error())
			return
		}
		writeJSON(w, http.StatusOK, versions)
	case http.MethodPost:
		var body struct {
			IndividualID string `json:"individual_id"`
			Name         string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.IndividualID == "" || body.Name == "" {
			writeError(w, http.StatusBadRequest, "individual_id 与 name 必填")
			return
		}
		v, err := s.svc.CreateVersion(body.IndividualID, body.Name)
		if err != nil {
			writeError(w, statusForError(err), err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, v)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleVersionByID 处理单个版本：GET 详情（含边成员）/ POST 追加·移除边或状态流转。
func (s *Server) handleVersionByID(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.URL.Path, "/api/versions/")
	if id == "" {
		writeError(w, http.StatusBadRequest, "缺少版本 ID")
		return
	}
	switch r.Method {
	case http.MethodGet:
		v, err := s.svc.DB().GetVersion(id)
		if err != nil {
			writeError(w, statusForError(err), err.Error())
			return
		}
		edgeIDs, err := s.svc.DB().ListVersionEdges(id)
		if err != nil {
			writeError(w, statusForError(err), err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"version": v, "edge_ids": edgeIDs})
	case http.MethodPost:
		action := parseAction(r.URL.Path, "/api/versions/"+id+"/")
		var body struct {
			EdgeID string `json:"edge_id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		var err error
		switch action {
		case "add-edge":
			err = s.svc.AddEdgeToVersion(id, body.EdgeID)
		case "remove-edge":
			err = s.svc.RemoveEdgeFromVersion(id, body.EdgeID)
		case "share":
			err = s.svc.TransitionVersion(id, model.VersionShared)
		case "freeze":
			err = s.svc.TransitionVersion(id, model.VersionFrozen)
		case "supersede":
			err = s.svc.TransitionVersion(id, model.VersionSuperseded)
		default:
			writeError(w, http.StatusBadRequest, "未知动作: "+action)
			return
		}
		if err != nil {
			writeError(w, statusForError(err), err.Error())
			return
		}
		v, _ := s.svc.DB().GetVersion(id)
		writeJSON(w, http.StatusOK, v)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

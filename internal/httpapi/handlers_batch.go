package httpapi

import (
	"encoding/json"
	"net/http"

	"task253-birdbanding/internal/model"
)

// handleBatches 处理批次集合：GET 列表 / POST 创建。
func (s *Server) handleBatches(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		batches, err := s.svc.DB().ListBatches()
		if err != nil {
			writeError(w, statusForError(err), err.Error())
			return
		}
		writeJSON(w, http.StatusOK, batches)
	case http.MethodPost:
		var body struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
			writeError(w, http.StatusBadRequest, "name 必填")
			return
		}
		b, err := s.svc.CreateBatch(body.Name)
		if err != nil {
			writeError(w, statusForError(err), err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, b)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleBatchByID 处理单个批次：GET 详情 / POST 流转（publish/seal）。
func (s *Server) handleBatchByID(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.URL.Path, "/api/batches/")
	if id == "" {
		writeError(w, http.StatusBadRequest, "缺少批次 ID")
		return
	}
	switch r.Method {
	case http.MethodGet:
		b, err := s.svc.DB().GetBatch(id)
		if err != nil {
			writeError(w, statusForError(err), err.Error())
			return
		}
		writeJSON(w, http.StatusOK, b)
	case http.MethodPost:
		action := parseAction(r.URL.Path, "/api/batches/"+id+"/")
		var to model.BatchStatus
		switch action {
		case "publish":
			to = model.BatchPublished
		case "seal":
			to = model.BatchSealed
		case "review":
			to = model.BatchReview
		default:
			writeError(w, http.StatusBadRequest, "未知动作: "+action)
			return
		}
		if err := s.svc.TransitionBatch(id, to); err != nil {
			writeError(w, statusForError(err), err.Error())
			return
		}
		b, _ := s.svc.DB().GetBatch(id)
		writeJSON(w, http.StatusOK, b)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// parseAction 从 /prefix/<id>/<action> 提取动作段。
func parseAction(path, prefix string) string {
	rest := path[len(prefix):]
	if i := indexByte(rest, '/'); i >= 0 {
		rest = rest[:i]
	}
	return rest
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

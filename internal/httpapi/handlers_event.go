package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"task253-birdbanding/internal/event"
	"task253-birdbanding/internal/model"
)

// handleEvents 处理事件集合：POST 导入单个 / POST /bulk 批量 / GET 列表。
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	// Bulk import keeps malformed observations isolated from valid siblings.
	switch r.Method {
	case http.MethodPost:
		var body struct {
			BatchID    string `json:"batch_id"`
			RingCode   string `json:"ring_code"`
			Type       string `json:"type"`
			LocationID string `json:"location_id"`
			EventDate  string `json:"event_date"`
			Species    string `json:"species"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "请求体解析失败")
			return
		}
		ed, err := time.Parse("2006-01-02", body.EventDate)
		if err != nil {
			writeError(w, http.StatusBadRequest, "event_date 格式应为 2006-01-02")
			return
		}
		ev, existed, err := s.svc.ImportEvent(event.ImportInput{
			BatchID:    body.BatchID,
			RingCode:   body.RingCode,
			Type:       model.EventType(body.Type),
			LocationID: body.LocationID,
			EventDate:  ed,
			Species:    body.Species,
		})
		if err != nil {
			writeError(w, statusForError(err), err.Error())
			return
		}
		if existed {
			writeJSON(w, http.StatusOK, map[string]interface{}{"event": ev, "existed": true})
			return
		}
		writeJSON(w, http.StatusCreated, ev)
	case http.MethodGet:
		q := r.URL.Query()
		events, err := s.svc.DB().ListEvents(q.Get("batch_id"), q.Get("individual_id"), q.Get("status"))
		if err != nil {
			writeError(w, statusForError(err), err.Error())
			return
		}
		writeJSON(w, http.StatusOK, events)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleEventByID 处理单个事件：GET 详情 / POST 校验·校正环号·排除。
func (s *Server) handleEventByID(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.URL.Path, "/api/events/")
	if id == "" {
		writeError(w, http.StatusBadRequest, "缺少事件 ID")
		return
	}
	switch r.Method {
	case http.MethodGet:
		ev, err := s.svc.DB().GetEvent(id)
		if err != nil {
			writeError(w, statusForError(err), err.Error())
			return
		}
		writeJSON(w, http.StatusOK, ev)
	case http.MethodPost:
		action := parseAction(r.URL.Path, "/api/events/"+id+"/")
		var body struct {
			RingCode    string `json:"ring_code"`
			Species     string `json:"species"`
			Reason      string `json:"reason"`
			BandingDate string `json:"banding_date"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		var err error
		switch action {
		case "validate":
			var bd time.Time
			if body.BandingDate != "" {
				bd, _ = time.Parse("2006-01-02", body.BandingDate)
			}
			err = s.svc.ValidateEvent(id, bd)
		case "correct-ring":
			err = s.svc.CorrectRing(id, body.RingCode, body.Species)
		case "exclude":
			err = s.svc.ExcludeEvent(id, body.Reason)
		default:
			writeError(w, http.StatusBadRequest, "未知动作: "+action)
			return
		}
		if err != nil {
			writeError(w, statusForError(err), err.Error())
			return
		}
		ev, _ := s.svc.DB().GetEvent(id)
		writeJSON(w, http.StatusOK, ev)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

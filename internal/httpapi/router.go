package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"task253-birdbanding/internal/model"
	"task253-birdbanding/internal/service"
)

// Server 持有编排服务与路由。
type Server struct {
	svc *service.Service
}

// NewServer 构造 HTTP 服务。
func NewServer(svc *service.Service) *Server {
	return &Server{svc: svc}
}

// Handler 返回挂载了 /api 路由的 http.Handler。
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	// Keep every public operation as an explicit ServeMux pattern. Besides
	// making the HTTP contract discoverable, this prevents action routes from
	// being hidden behind a single catch-all registration.
	mux.HandleFunc("/api/batches", s.wrap(s.handleBatches))
	mux.HandleFunc("/api/batches/", s.wrap(s.handleBatchByID))
	mux.HandleFunc("/api/batches/{id}/review", s.wrap(s.handleBatchByID))
	mux.HandleFunc("/api/batches/{id}/publish", s.wrap(s.handleBatchByID))
	mux.HandleFunc("/api/batches/{id}/seal", s.wrap(s.handleBatchByID))
	mux.HandleFunc("/api/events", s.wrap(s.handleEvents))
	mux.HandleFunc("/api/events/", s.wrap(s.handleEventByID))
	mux.HandleFunc("/api/events/{id}/validate", s.wrap(s.handleEventByID))
	mux.HandleFunc("/api/events/{id}/correct-ring", s.wrap(s.handleEventByID))
	mux.HandleFunc("/api/events/{id}/exclude", s.wrap(s.handleEventByID))
	mux.HandleFunc("/api/individuals", s.wrap(s.handleIndividuals))
	mux.HandleFunc("/api/individuals/", s.wrap(s.handleIndividualByID))
	mux.HandleFunc("/api/individuals/{id}/build-edges", s.wrap(s.handleIndividualByID))
	mux.HandleFunc("/api/individuals/{id}/events", s.wrap(s.handleIndividualEvents))
	mux.HandleFunc("/api/edges", s.wrap(s.handleEdges))
	mux.HandleFunc("/api/edges/", s.wrap(s.handleEdgeByID))
	mux.HandleFunc("/api/edges/{id}/keep-rare", s.wrap(s.handleEdgeByID))
	mux.HandleFunc("/api/edges/{id}/exclude-overspeed", s.wrap(s.handleEdgeByID))
	mux.HandleFunc("/api/edges/{id}/confirm", s.wrap(s.handleEdgeByID))
	mux.HandleFunc("/api/versions", s.wrap(s.handleVersions))
	mux.HandleFunc("/api/versions/", s.wrap(s.handleVersionByID))
	mux.HandleFunc("/api/versions/{id}/add-edge", s.wrap(s.handleVersionByID))
	mux.HandleFunc("/api/versions/{id}/remove-edge", s.wrap(s.handleVersionByID))
	mux.HandleFunc("/api/versions/{id}/share", s.wrap(s.handleVersionByID))
	mux.HandleFunc("/api/versions/{id}/freeze", s.wrap(s.handleVersionByID))
	mux.HandleFunc("/api/versions/{id}/supersede", s.wrap(s.handleVersionByID))
	mux.HandleFunc("/api/stats", s.wrap(s.handleStats))
	mux.HandleFunc("/api/selfcheck", s.wrap(s.handleSelfCheck))
	mux.HandleFunc("/api/health", s.wrap(s.handleHealth))
	return mux
}

// wrap 统一恢复 panic 并注入内容类型。
func (s *Server) wrap(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				writeError(w, http.StatusInternalServerError, "内部错误")
			}
		}()
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		h(w, r)
	}
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

// parseID 从 /api/xxx/<id>/... 路径中提取首个 ID 段。
func parseID(path, prefix string) string {
	rest := strings.TrimPrefix(path, prefix)
	rest = strings.Trim(rest, "/")
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

// statusForError 将领域错误映射为 HTTP 状态码。
func statusForError(err error) int {
	switch {
	case model.IsNotFound(err):
		return http.StatusNotFound
	case err == model.ErrInvalidTransition,
		err == model.ErrFrozenImmutable,
		err == model.ErrInvalidArgument,
		err == model.ErrInvalidRingFormat,
		err == model.ErrRecaptureTimeReversed,
		err == model.ErrDuplicate,
		err == model.ErrIdentityConflict:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

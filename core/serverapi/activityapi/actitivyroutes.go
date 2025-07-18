package activityapi

import (
	"net/http"
	"strconv"

	"github.com/contenox/runtime-mvp/core/serverops"
	"github.com/contenox/runtime-mvp/core/services/activityservice"
	"github.com/contenox/runtime-mvp/core/taskengine"
)

func AddActivityRoutes(mux *http.ServeMux, _ *serverops.Config, activityService activityservice.Service) {
	s := &activityAPI{service: activityService}
	mux.HandleFunc("GET /activity/logs", s.list)
	mux.HandleFunc("GET /activity/requests", s.requests)
	mux.HandleFunc("GET /activity/requests/{id}", s.requestByID)
	mux.HandleFunc("GET /activity/requests/{id}/state", s.getExecutionState)
	mux.HandleFunc("GET /activity/operations", s.operations)
	mux.HandleFunc("GET /activity/operations/{op}/{subject}", s.requestsByOperation)
	mux.HandleFunc("GET /activity/stateful-requests", s.getStatefulRequests)
	mux.HandleFunc("GET /activity/alerts", s.alerts)
}

type activityAPI struct {
	service activityservice.Service
}

func (s *activityAPI) list(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse ?limit=N from query (default: 100)
	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	logs, err := s.service.GetLogs(ctx, limit)
	if err != nil {
		_ = serverops.Error(w, r, err, serverops.ListOperation)
		return
	}

	serverops.Encode(w, r, http.StatusOK, logs)
}

func (s *activityAPI) requests(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	requests, err := s.service.GetRequests(ctx, limit)
	if err != nil {
		_ = serverops.Error(w, r, err, serverops.ListOperation)
		return
	}

	serverops.Encode(w, r, http.StatusOK, requests)
}

func (s *activityAPI) requestByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	reqID := r.PathValue("id")
	events, err := s.service.GetRequest(ctx, reqID)
	if err != nil {
		_ = serverops.Error(w, r, err, serverops.GetOperation)
		return
	}

	serverops.Encode(w, r, http.StatusOK, events)
}

func (s *activityAPI) operations(w http.ResponseWriter, r *http.Request) {
	ops, err := s.service.GetKnownOperations(r.Context())
	if err != nil {
		_ = serverops.Error(w, r, err, serverops.ListOperation)
		return
	}
	serverops.Encode(w, r, http.StatusOK, ops)
}

func (s *activityAPI) requestsByOperation(w http.ResponseWriter, r *http.Request) {
	op := taskengine.Operation{
		Operation: r.PathValue("op"),
		Subject:   r.PathValue("subject"),
	}
	reqs, err := s.service.GetRequestIDByOperation(r.Context(), op)
	if err != nil {
		_ = serverops.Error(w, r, err, serverops.ListOperation)
		return
	}
	serverops.Encode(w, r, http.StatusOK, reqs)
}

func (s *activityAPI) getExecutionState(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqID := r.PathValue("id")

	state, err := s.service.GetExecutionState(ctx, reqID)
	if err != nil {
		_ = serverops.Error(w, r, err, serverops.GetOperation)
		return
	}

	serverops.Encode(w, r, http.StatusOK, state)
}

func (s *activityAPI) getStatefulRequests(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqIDs, err := s.service.GetStatefulRequests(ctx)
	if err != nil {
		_ = serverops.Error(w, r, err, serverops.ListOperation)
		return
	}
	serverops.Encode(w, r, http.StatusOK, reqIDs)
}

func (s *activityAPI) alerts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	limit := 99
	limitStr := r.URL.Query().Get("limit")
	if limitStr != "" {
		limit, _ = strconv.Atoi(limitStr)
	}

	alerts, err := s.service.FetchAlerts(ctx, limit)
	if err != nil {
		_ = serverops.Error(w, r, err, serverops.ListOperation)
		return
	}
	serverops.Encode(w, r, http.StatusOK, alerts)
}

package execapi

import (
	"errors"
	"net/http"

	"github.com/contenox/runtime-mvp/core/serverops"
	"github.com/contenox/runtime-mvp/core/services/execservice"
	"github.com/contenox/runtime-mvp/core/taskengine"
)

func AddExecRoutes(mux *http.ServeMux, _ *serverops.Config, promptService execservice.ExecService, taskService execservice.TasksEnvService) {
	f := &taskManager{
		promptService: promptService,
		taskService:   taskService,
	}
	mux.HandleFunc("POST /execute", f.execute)
	mux.HandleFunc("POST /tasks/attach/connector/{id}", f.attachToConnector)
	mux.HandleFunc("POST /tasks", f.tasks)
	mux.HandleFunc("GET /supported", f.supported)
}

type taskManager struct {
	promptService execservice.ExecService
	taskService   execservice.TasksEnvService
}

func (tm *taskManager) execute(w http.ResponseWriter, r *http.Request) {
	req, err := serverops.Decode[execservice.TaskRequest](r)
	if err != nil {
		_ = serverops.Error(w, r, err, serverops.ExecuteOperation)
		return
	}

	resp, err := tm.promptService.Execute(r.Context(), &req)
	if err != nil {
		_ = serverops.Error(w, r, err, serverops.ExecuteOperation)
		return
	}
	_ = serverops.Encode(w, r, http.StatusOK, resp)
}

type taskExec struct {
	Input string                      `json:"input"`
	Chain *taskengine.ChainDefinition `json:"chain"`
}

func (tm *taskManager) tasks(w http.ResponseWriter, r *http.Request) {
	req, err := serverops.Decode[taskExec](r)
	if err != nil {
		_ = serverops.Error(w, r, err, serverops.ExecuteOperation)
		return
	}

	resp, capturedStateUnits, err := tm.taskService.Execute(r.Context(), req.Chain, req.Input)
	if err != nil {
		_ = serverops.Error(w, r, err, serverops.ExecuteOperation)
		return
	}
	response := map[string]any{
		"response": resp,
		"state":    capturedStateUnits,
	}
	_ = serverops.Encode(w, r, http.StatusOK, response)
}

func (tm *taskManager) attachToConnector(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		_ = serverops.Error(w, r, errors.New("missing id"), serverops.ExecuteOperation)
		return
	}
	req, err := serverops.Decode[taskExec](r)
	if err != nil {
		_ = serverops.Error(w, r, err, serverops.ExecuteOperation)
		return
	}

	err = tm.taskService.AttachToConnector(r.Context(), id, req.Chain)
	if err != nil {
		_ = serverops.Error(w, r, err, serverops.ExecuteOperation)
		return
	}
	_ = serverops.Encode(w, r, http.StatusOK, map[string]string{"message": "taskchain was attached"})
}

func (tm *taskManager) supported(w http.ResponseWriter, r *http.Request) {
	resp, err := tm.taskService.Supports(r.Context())
	if err != nil {
		_ = serverops.Error(w, r, err, serverops.ListOperation)
		return
	}

	_ = serverops.Encode(w, r, http.StatusOK, resp)
}

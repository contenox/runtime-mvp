package execservice

import (
	"context"

	"github.com/contenox/runtime-mvp/core/serverops"
	"github.com/contenox/runtime-mvp/core/serverops/store"
	"github.com/contenox/runtime-mvp/core/taskengine"
	"github.com/contenox/runtime-mvp/libs/libdb"
)

type TasksEnvService interface {
	Execute(ctx context.Context, chain *taskengine.ChainDefinition, input string) (any, []taskengine.CapturedStateUnit, error)
	AttachToConnector(ctx context.Context, connectorID string, chain *taskengine.ChainDefinition) error
	serverops.ServiceMeta
	taskengine.HookRegistry
}

type tasksEnvService struct {
	environmentExec taskengine.EnvExecutor
	db              libdb.DBManager
	hookRegistry    taskengine.HookRegistry
}

func NewTasksEnv(ctx context.Context, environmentExec taskengine.EnvExecutor, dbInstance libdb.DBManager, hookRegistry taskengine.HookRegistry) TasksEnvService {
	return &tasksEnvService{
		environmentExec: environmentExec,
		db:              dbInstance,
		hookRegistry:    hookRegistry,
	}
}

func (s *tasksEnvService) Execute(ctx context.Context, chain *taskengine.ChainDefinition, input string) (any, []taskengine.CapturedStateUnit, error) {
	tx := s.db.WithoutTransaction()

	storeInstance := store.New(tx)
	// TODO: check permission view? why not exec?
	if err := serverops.CheckServiceAuthorization(ctx, storeInstance, s, store.PermissionView); err != nil {
		return nil, nil, err
	}

	return s.environmentExec.ExecEnv(ctx, chain, input, taskengine.DataTypeAny)
}

// AttachToConnector implements TasksEnvService.
func (s *tasksEnvService) AttachToConnector(ctx context.Context, connectorID string, chain *taskengine.ChainDefinition) error {
	panic("unimplemented")
}

func (s *tasksEnvService) GetServiceName() string {
	return "taskenviromentservice"
}

func (s *tasksEnvService) GetServiceGroup() string {
	return serverops.DefaultDefaultServiceGroup
}

func (s *tasksEnvService) Supports(ctx context.Context) ([]string, error) {
	return s.hookRegistry.Supports(ctx)
}

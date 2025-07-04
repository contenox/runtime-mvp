package tasksrecipes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/contenox/runtime-mvp/core/serverops"
	"github.com/contenox/runtime-mvp/core/serverops/store"
	"github.com/contenox/runtime-mvp/core/taskengine"
	"github.com/contenox/runtime-mvp/libs/libdb"
)

const (
	OpenAIChatChainID   = "openai_chat_chain"
	StandardChatChainID = "chat_chain"
)

func initializeDefaultChains(ctx context.Context, cfg *serverops.Config, db libdb.DBManager) error {
	// Create chains with proper IDs
	chains := []*taskengine.ChainDefinition{
		BuildOpenAIChatChain(cfg.TasksModel, "ollama"),
		BuildChatChain(BuildChatChainReq{
			PreferredModelNames: []string{cfg.TasksModel},
			Provider:            "ollama",
		}),
	}
	tx, comm, end, err := db.WithTransaction(ctx)
	defer end()
	if err != nil {
		return err
	}
	// Store chains
	for _, chain := range chains {
		var value any
		err := store.New(tx).GetKV(ctx, chain.ID, &value)
		if err != nil && !errors.Is(err, libdb.ErrNotFound) {
			return fmt.Errorf("failed to retrieve chain %s: %v", chain.ID, err)
		}
		if errors.Is(err, libdb.ErrNotFound) {
			if err := SetChainDefinition(ctx, tx, chain); err != nil {
				log.Printf("failed to initialize chain %s: %v", chain.ID, err)
			}
		}
	}
	return comm(ctx)
}

func BuildOpenAIChatChain(model string, llmProvider string) *taskengine.ChainDefinition {
	return &taskengine.ChainDefinition{
		ID:          "openai_chat_chain",
		Description: "OpenAI Style chat processing pipeline with hooks",
		Tasks: []taskengine.ChainTask{
			{
				ID:          "convert_openai_to_history",
				Description: "Convert OpenAI request to internal history",
				Type:        taskengine.Hook,
				Hook: &taskengine.HookCall{
					Type: "convert_openai_to_history",
					Args: map[string]string{},
				},
				Transition: taskengine.TaskTransition{
					Branches: []taskengine.TransitionBranch{
						{Operator: "default", Goto: "execute_model_on_messages"},
					},
				},
			},
			{
				ID:          "execute_model_on_messages",
				Description: "Run inference using selected LLM",
				Type:        taskengine.Hook,
				Transition: taskengine.TaskTransition{
					Branches: []taskengine.TransitionBranch{
						{Operator: "default", Goto: "convert_history_to_openai"},
					},
				},
				Hook: &taskengine.HookCall{
					Type: "execute_model_on_messages",
					Args: map[string]string{
						"model":    model,
						"provider": llmProvider,
					},
				},
			},
			{
				ID:          "convert_history_to_openai",
				Description: "Convert chat history to OpenAI response",
				Type:        taskengine.Hook,
				Hook: &taskengine.HookCall{
					Type: "convert_history_to_openai",
					Args: map[string]string{
						"model": model,
					},
				},
				Transition: taskengine.TaskTransition{
					Branches: []taskengine.TransitionBranch{
						{Operator: "default", Goto: taskengine.TermEnd},
					},
				},
			},
		},
	}
}

type BuildChatChainReq struct {
	SubjectID           string
	PreferredModelNames []string
	Provider            string
}

func BuildChatChain(req BuildChatChainReq) *taskengine.ChainDefinition {
	return &taskengine.ChainDefinition{
		ID:          "chat_chain",
		Description: "Standard chat processing pipeline with hooks",
		Tasks: []taskengine.ChainTask{
			{
				ID:          "append_user_message",
				Description: "Append user message to chat history",
				Type:        taskengine.Hook,
				Hook: &taskengine.HookCall{
					Type: "append_user_message",
					Args: map[string]string{
						"subject_id": req.SubjectID,
					},
				},
				Transition: taskengine.TaskTransition{
					Branches: []taskengine.TransitionBranch{
						{Operator: "default", Goto: "preappend_message_to_history"},
					},
				},
			},
			{
				ID:          "preappend_message_to_history",
				Description: "Add system level instructions to chat history",
				Type:        taskengine.Hook,
				Hook: &taskengine.HookCall{
					Type: "preappend_message_to_history",
					Args: map[string]string{
						"role":    "system",
						"message": "You are a helpful assistant. Part of a larger system named \"contenox\".",
					},
				},
				Transition: taskengine.TaskTransition{
					Branches: []taskengine.TransitionBranch{
						{Operator: "default", Goto: "mux_input"},
					},
				},
			},
			{
				ID:          "mux_input",
				Description: "Check for commands like /echo using Mux",
				Type:        taskengine.Hook,
				Hook: &taskengine.HookCall{
					Type: "command_router",
					Args: map[string]string{
						"subject_id": req.SubjectID,
					},
				},
				Transition: taskengine.TaskTransition{
					Branches: []taskengine.TransitionBranch{
						{Operator: "default", Goto: "execute_model_on_messages"},
						{
							Operator: "equals",
							When:     "echo",
							Goto:     "persist_messages",
						},
					},
				},
			},
			{
				ID:          "execute_model_on_messages",
				Description: "Run inference using selected LLM",
				Type:        taskengine.Hook,
				Transition: taskengine.TaskTransition{
					Branches: []taskengine.TransitionBranch{
						{Operator: "default", Goto: "persist_messages"},
					},
				},
				Hook: &taskengine.HookCall{
					Type: "execute_model_on_messages",
					Args: map[string]string{
						"subject_id": req.SubjectID,
						"models":     strings.Join(req.PreferredModelNames, ","),
						"provider":   req.Provider,
					},
				},
			},
			{
				ID:          "persist_messages",
				Description: "Persist the conversation",
				Type:        taskengine.Hook,
				Hook: &taskengine.HookCall{
					Type: "persist_messages",
					Args: map[string]string{
						"subject_id": req.SubjectID,
					},
				},
				Transition: taskengine.TaskTransition{
					Branches: []taskengine.TransitionBranch{
						{Operator: "default", Goto: taskengine.TermEnd},
					},
				},
			},
		},
	}
}

func BuildAppendInstruction(subjectID string) *taskengine.ChainDefinition {
	return &taskengine.ChainDefinition{
		Tasks: []taskengine.ChainTask{
			{
				ID:          "append_system_message",
				Description: "Append instruction message to chat history",
				Type:        taskengine.Hook,
				Hook: &taskengine.HookCall{
					Type: "append_system_message",
					Args: map[string]string{
						"subject_id": subjectID,
					},
				},
				Transition: taskengine.TaskTransition{
					Branches: []taskengine.TransitionBranch{
						{Operator: "default", Goto: taskengine.TermEnd},
					},
				},
			},
		},
	}
}

const ChainKeyPrefix = "chain:"

func SetChainDefinition(ctx context.Context, tx libdb.Exec, chain *taskengine.ChainDefinition) error {
	s := store.New(tx)
	key := ChainKeyPrefix + chain.ID
	data, err := json.Marshal(chain)
	if err != nil {
		return err
	}
	return s.SetKV(ctx, key, data)
}

func GetChainDefinition(ctx context.Context, tx libdb.Exec, id string) (*taskengine.ChainDefinition, error) {
	s := store.New(tx)
	key := ChainKeyPrefix + id
	var chain taskengine.ChainDefinition
	if err := s.GetKV(ctx, key, &chain); err != nil {
		return nil, err
	}
	return &chain, nil
}

func ListChainDefinitions(ctx context.Context, tx libdb.Exec) ([]*taskengine.ChainDefinition, error) {
	s := store.New(tx)
	kvs, err := s.ListKVPrefix(ctx, ChainKeyPrefix)
	if err != nil {
		return nil, err
	}

	chains := make([]*taskengine.ChainDefinition, 0, len(kvs))
	for _, kv := range kvs {
		var chain taskengine.ChainDefinition
		if err := json.Unmarshal(kv.Value, &chain); err != nil {
			return nil, err
		}
		chains = append(chains, &chain)
	}
	return chains, nil
}

func DeleteChainDefinition(ctx context.Context, tx libdb.Exec, id string) error {
	s := store.New(tx)
	key := ChainKeyPrefix + id
	return s.DeleteKV(ctx, key)
}

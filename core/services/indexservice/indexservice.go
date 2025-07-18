package indexservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/contenox/runtime-mvp/core/indexrepo"
	"github.com/contenox/runtime-mvp/core/llmrepo"
	"github.com/contenox/runtime-mvp/core/llmresolver"
	"github.com/contenox/runtime-mvp/core/serverops"
	"github.com/contenox/runtime-mvp/core/serverops/store"
	"github.com/contenox/runtime-mvp/core/serverops/vectors"
	"github.com/contenox/runtime-mvp/libs/libdb"
)

type Service interface {
	Index(ctx context.Context, request *IndexRequest) (*IndexResponse, error)
	Search(ctx context.Context, request *SearchRequest) (*SearchResponse, error)
	ListKeywords(ctx context.Context) ([]string, error)
	serverops.ServiceMeta
}

type service struct {
	embedder     llmrepo.ModelRepo
	promptExec   llmrepo.ModelRepo
	vectorsStore vectors.Store
	db           libdb.DBManager
}

func New(ctx context.Context, embedder, promptExec llmrepo.ModelRepo, vectorsStore vectors.Store, dbInstance libdb.DBManager) Service {
	return &service{
		embedder:     embedder,
		promptExec:   promptExec,
		vectorsStore: vectorsStore,
		db:           dbInstance,
	}
}

type IndexRequest struct {
	Chunks   []string `json:"chunks"`
	ID       string   `json:"id"`
	Replace  bool     `json:"replace"`
	JobID    string   `json:"jobId"`
	LeaserID string   `json:"leaserId"`
}

type Keyword struct {
	Text string `json:"text"`
}

type IndexResponse struct {
	ID                string   `json:"id"`
	VectorIDs         []string `json:"vectors"`
	AugmentedMetadata []string `json:"augmentedMetadata"`
}

func (s *service) Index(ctx context.Context, request *IndexRequest) (*IndexResponse, error) {
	if request.LeaserID == "" {
		return nil, serverops.ErrMissingParameter
	}
	if request.JobID == "" {
		return nil, serverops.ErrMissingParameter
	}

	var vectorIDs []string

	clean := func() {
		for _, id := range vectorIDs {
			if delErr := s.vectorsStore.Delete(ctx, id); delErr != nil {
				fmt.Printf("failed to clean up vector %q: %v\n", id, delErr)
			}
		}
	}

	tx, commit, end, err := s.db.WithTransaction(ctx, clean)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer end()

	storeInstance := store.New(tx)

	if err := serverops.CheckServiceAuthorization(ctx, storeInstance, s, store.PermissionEdit); err != nil {
		return nil, err
	}

	job, err := storeInstance.GetLeasedJob(ctx, request.JobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get leased job %s: %w", request.JobID, err)
	}
	if job.Leaser != request.LeaserID {
		return nil, fmt.Errorf("job is not leased by this leaser")
	}

	// Replace existing vectors if needed
	if request.Replace {
		chunks, err := storeInstance.ListChunkIndicesByResource(ctx, request.ID, store.ResourceTypeFile)
		if err != nil {
			return nil, fmt.Errorf("failed to get chunk index by ID: %w", err)
		}
		for _, chunk := range chunks {
			err := s.vectorsStore.Delete(ctx, chunk.VectorID)
			if err != nil {
				return nil, fmt.Errorf("failed to delete vector: %w", err)
			}
			if delErr := storeInstance.DeleteKV(ctx, "vector:"+chunk.VectorID); delErr != nil && errors.Is(delErr, libdb.ErrNotFound) {
				return nil, fmt.Errorf("failed to delete KV for vector %s: %w", chunk.VectorID, delErr)
			}
		}
	}

	augmentStrategy := func(ctx context.Context, resourceID string, vectorID string, chunk string) (string, error) {
		dbInstance := store.New(s.db.WithoutTransaction())
		keywords, err := s.findKeywords(ctx, chunk)
		if err != nil {
			return "", fmt.Errorf("failed to enrich chunk: %w", err)
		}

		keywordsJSON, err := json.Marshal(Keyword{Text: keywords})
		if err != nil {
			return "", fmt.Errorf("failed to marshal keywords: %w", err)
		}
		err = dbInstance.SetKV(ctx, "vector:"+vectorID, keywordsJSON)
		if err != nil {
			return "", fmt.Errorf("failed to set keywords: %w", err)
		}

		return fmt.Sprintf("%s\n\nKeywords: %s", chunk, keywords), nil
	}

	vectorIDs, augmentedMetadata, err := indexrepo.IngestChunks(
		ctx,
		s.embedder,
		s.vectorsStore,
		tx,
		request.ID,
		job.EntityType,
		request.Chunks,
		augmentStrategy,
	)
	if err != nil {
		return nil, err
	}

	err = storeInstance.DeleteLeasedJob(ctx, job.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete leased job: %w", err)
	}

	err = commit(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to commit: %w", err)
	}

	return &IndexResponse{
		ID:                request.ID,
		VectorIDs:         vectorIDs,
		AugmentedMetadata: augmentedMetadata,
	}, nil
}

type SearchRequest struct {
	Query       string `json:"text"`
	TopK        int    `json:"topK"`
	ExpandFiles bool   `json:"expandFiles"`
	*SearchRequestArgs
}

type SearchRequestArgs struct {
	Epsilon float32 `json:"epsilon"`
	Radius  float32 `json:"radius"`
}

type SearchResponse struct {
	Results      []indexrepo.SearchResult `json:"results"`
	TriedQueries []string                 `json:"triedQuery"`
}

func (s *service) Search(ctx context.Context, request *SearchRequest) (*SearchResponse, error) {
	tx := s.db.WithoutTransaction()
	storeInstance := store.New(tx)
	if err := serverops.CheckServiceAuthorization(ctx, storeInstance, s, store.PermissionView); err != nil {
		return nil, err
	}
	tryQuery := []string{request.Query}

	isQuestion, err := s.classifyQuestion(ctx, request.Query)
	if err != nil {
		return nil, fmt.Errorf("question classification failed: %w", err)
	}
	if isQuestion {
		stripedQuery, err := s.convertQuestionQuery(ctx, request.Query)
		if err != nil {
			return nil, fmt.Errorf("question rewriteQuery failed: %w", err)
		}
		tryQuery = append(tryQuery, stripedQuery)
	}

	topK := request.TopK
	if topK <= 0 {
		topK = 10
	}
	var args *indexrepo.Args
	if request.SearchRequestArgs != nil {
		args = &indexrepo.Args{
			Epsilon: request.SearchRequestArgs.Epsilon,
			Radius:  request.SearchRequestArgs.Radius,
		}
	}
	searchResults, err := indexrepo.ExecuteVectorSearch(
		ctx,
		s.embedder,
		s.vectorsStore,
		tx,
		tryQuery,
		topK,
		args,
	)
	if err != nil {
		return nil, err
	}

	if request.ExpandFiles {
		for i, sr := range searchResults {
			if sr.ResourceType == store.ResourceTypeFile {
				file, err := storeInstance.GetFileByID(ctx, sr.ID)
				if err != nil {
					return nil, fmt.Errorf("BADSERVER STATE %w", err)
				}
				searchResults[i].FileMeta = file
			}
			if delKVErr := storeInstance.DeleteKV(ctx, "vector:"+sr.ChunkID); delKVErr != nil && errors.Is(delKVErr, libdb.ErrNotFound) {
				return nil, fmt.Errorf("BADSERVER STATE failed to clean orphaned KV %s: %v", sr.ID, delKVErr)
			}
			continue
		}
	}

	return &SearchResponse{
		Results:      searchResults,
		TriedQueries: tryQuery,
	}, nil
}

func (s *service) findKeywords(ctx context.Context, chunk string) (string, error) {
	prompt := fmt.Sprintf(`Extract 5-7 keywords from the following text:

	%s

	Return a comma-separated list of keywords.`, chunk)

	provider, err := s.promptExec.GetDefaultSystemProvider(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get provider: %w", err)
	}

	promptClient, err := llmresolver.PromptExecute(ctx, llmresolver.PromptRequest{
		ModelNames: []string{provider.ModelName()},
	}, s.promptExec.GetRuntime(ctx), llmresolver.Randomly)
	if err != nil {
		return "", fmt.Errorf("failed to resolve prompt client for model %s: %w", provider.ModelName(), err)
	}
	response, err := promptClient.Prompt(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("failed to execute the prompt: %w", err)
	}
	return response, nil
}

func (s *service) classifyQuestion(ctx context.Context, input string) (bool, error) {
	prompt := fmt.Sprintf(`Analyze if the following input is a question? Answer strictly with "yes" or "no".

	Input: %s`, input)

	response, err := s.executePrompt(ctx, prompt)
	if err != nil {
		return false, err
	}

	return strings.EqualFold(strings.TrimSpace(response), "yes"), nil
}

func (s *service) convertQuestionQuery(ctx context.Context, query string) (string, error) {
	promptTemplate := `Convert the following question into a search query using exactly the original keywords by removing question words.

	Input: %s

	Optimized query:`

	prompt := fmt.Sprintf(promptTemplate, query)
	return s.executePrompt(ctx, prompt)
}

func (s *service) executePrompt(ctx context.Context, prompt string) (string, error) {
	provider, err := s.promptExec.GetDefaultSystemProvider(ctx)
	if err != nil {
		return "", fmt.Errorf("provider resolution failed: %w", err)
	}

	client, err := llmresolver.PromptExecute(ctx, llmresolver.PromptRequest{
		ModelNames: []string{provider.ModelName()},
	}, s.promptExec.GetRuntime(ctx), llmresolver.Randomly)
	if err != nil {
		return "", fmt.Errorf("client resolution failed: %w", err)
	}

	response, err := client.Prompt(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("prompt execution failed: %w", err)
	}

	return strings.TrimSpace(response), nil
}

func (s *service) ListKeywords(ctx context.Context) ([]string, error) {
	tx := s.db.WithoutTransaction()
	storeInstance := store.New(tx)

	if err := serverops.CheckServiceAuthorization(ctx, storeInstance, s, store.PermissionView); err != nil {
		return nil, err
	}

	// List all KV entries with "vector:" prefix
	kvs, err := storeInstance.ListKVPrefix(ctx, "vector:")
	if err != nil {
		return nil, fmt.Errorf("failed to list keywords: %w", err)
	}

	keywords := make([]string, 0, len(kvs))
	for _, kv := range kvs {
		var keyword Keyword
		if err := json.Unmarshal(kv.Value, &keyword); err != nil {
			return nil, fmt.Errorf("failed to unmarshal keyword (skipping): %w", err)
		}
		if keyword.Text != "" {
			keywords = append(keywords, keyword.Text)
		}
	}

	return keywords, nil
}

func (s *service) GetServiceName() string {
	return "indexservice"
}

func (s *service) GetServiceGroup() string {
	return serverops.DefaultDefaultServiceGroup // TODO: is that accurate?
}

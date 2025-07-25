package store

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/contenox/runtime-mvp/libs/libauth"
	"github.com/contenox/runtime-mvp/libs/libdb"
	"github.com/stretchr/testify/require"
)

type TelegramFrontend struct {
	ID            string     `json:"id"`
	UserID        string     `json:"userId"`
	ChatChain     string     `json:"chatChain"`
	Description   string     `json:"description"`
	BotToken      string     `json:"botToken"`
	SyncInterval  int        `json:"syncInterval"`
	Status        string     `json:"status"`
	LastOffset    int        `json:"lastOffset"`
	LastHeartbeat *time.Time `json:"lastHeartbeat"`
	LastError     string     `json:"lastError"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

type GitHubRepo struct {
	ID          string    `json:"id"`
	UserID      string    `json:"userId"`
	Owner       string    `json:"owner"`
	RepoName    string    `json:"repoName"`
	AccessToken string    `json:"accessToken"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Status struct {
	Status    string `json:"status"`
	Digest    string `json:"digest,omitempty"`
	Total     int64  `json:"total,omitempty"`
	Completed int64  `json:"completed,omitempty"`
	Model     string `json:"model"`
	BaseURL   string `json:"baseUrl"`
}

type QueueItem struct {
	URL   string `json:"url"`
	Model string `json:"model"`
}

type Backend struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	BaseURL string `json:"baseUrl"`
	Type    string `json:"type"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Message struct {
	ID      string    `json:"id"`
	IDX     string    `json:"stream"`
	Payload []byte    `json:"payload"`
	AddedAt time.Time `json:"addedAt"`
}

type Model struct {
	ID        string    `json:"id"`
	Model     string    `json:"model"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Pool struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	PurposeType string `json:"purposeType"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type User struct {
	ID               string `json:"id"`
	FriendlyName     string `json:"friendlyName"`
	Email            string `json:"email"`
	Subject          string `json:"subject"`
	HashedPassword   string `json:"hashedPassword"`
	RecoveryCodeHash string `json:"recoveryCodeHash"`
	Salt             string `json:"salt"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Job struct {
	ID           string    `json:"id"`
	TaskType     string    `json:"taskType"`
	Operation    string    `json:"operation"`
	Subject      string    `json:"subject"`
	EntityID     string    `json:"entityId"`
	EntityType   string    `json:"entityType"`
	Payload      []byte    `json:"payload"`
	ScheduledFor int64     `json:"scheduledFor"`
	ValidUntil   int64     `json:"validUntil"`
	RetryCount   int       `json:"retryCount"`
	CreatedAt    time.Time `json:"createdAt"`
}

type LeasedJob struct {
	Job
	Leaser          string    `json:"leaser"`
	LeaseExpiration time.Time `json:"leaseExpiration"`
}

type Resource struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

const (
	ResourceTypeSystem = "system"
	ResourceTypeFiles  = "files"
	ResourceTypeFile   = "file"
	ResourceTypeBlobs  = "blobs"
	ResourceTypeChunks = "chunks"
)

var ResourceTypes = []string{
	ResourceTypeSystem,
	ResourceTypeFiles,
	ResourceTypeBlobs,
	ResourceTypeChunks,
}

type File struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Meta      []byte    `json:"meta"`
	IsFolder  bool      `json:"isFolder"`
	BlobsID   string    `json:"blobsId"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Blob struct {
	ID        string    `json:"id"`
	Meta      []byte    `json:"meta"`
	Data      []byte    `json:"data"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type ChunkIndex struct {
	ID             string `json:"id"`
	VectorID       string `json:"vectorId"`
	VectorStore    string `json:"vectorStore"`
	ResourceID     string `json:"resourceId"`
	ResourceType   string `json:"resourceType"`
	EmbeddingModel string `json:"embeddingModel"`
}

type Permission int

const (
	PermissionNone Permission = iota
	PermissionView
	PermissionEdit
	PermissionManage
)

var permissionNames = map[Permission]string{
	PermissionNone:   "none",
	PermissionView:   "view",
	PermissionEdit:   "edit",
	PermissionManage: "manage",
}

var permissionValues = map[string]Permission{
	"none":   PermissionNone,
	"view":   PermissionView,
	"edit":   PermissionEdit,
	"manage": PermissionManage,
}

func (p Permission) String() string {
	if name, exists := permissionNames[p]; exists {
		return name
	}
	return "unknown"
}

func PermissionFromString(s string) (Permission, error) {
	if perm, exists := permissionValues[strings.ToLower(s)]; exists {
		return perm, nil
	}
	return PermissionNone, errors.New("invalid permission string")
}

type AccessEntry struct {
	ID           string     `json:"id"`
	Identity     string     `json:"identity"`
	Resource     string     `json:"resource"`
	ResourceType string     `json:"resourceType"`
	Permission   Permission `json:"permission"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

type AccessList []*AccessEntry

var ErrAccessEntryNotFound = errors.New("access denied")

var _ libauth.Authz = AccessList{}

func (al AccessList) RequireAuthorisation(forResource string, permission int) (bool, error) {
	found := false
	for _, entry := range al {
		if entry.Resource == forResource && entry.Permission >= Permission(permission) {
			return true, nil
		}
		if entry.Resource == forResource {
			found = true
		}
	}
	if !found {
		return false, ErrAccessEntryNotFound
	}
	return false, nil
}

// KV represents a key-value pair in the database
type KV struct {
	Key       string          `json:"key"`
	Value     json.RawMessage `json:"value"`
	CreatedAt time.Time       `json:"createdAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
}

type Bot struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	UserID string `json:"userId"`

	BotType     string    `json:"botType"`
	JobType     string    `json:"jobType"`
	State       []byte    `json:"state"`
	TaskChainID string    `json:"taskChainId"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Store interface {
	CreateBackend(ctx context.Context, backend *Backend) error
	GetBackend(ctx context.Context, id string) (*Backend, error)
	UpdateBackend(ctx context.Context, backend *Backend) error
	DeleteBackend(ctx context.Context, id string) error
	ListBackends(ctx context.Context) ([]*Backend, error)
	GetBackendByName(ctx context.Context, name string) (*Backend, error)

	AppendModel(ctx context.Context, model *Model) error
	GetModel(ctx context.Context, id string) (*Model, error)
	GetModelByName(ctx context.Context, name string) (*Model, error)
	DeleteModel(ctx context.Context, modelName string) error
	ListModels(ctx context.Context) ([]*Model, error)

	CreatePool(ctx context.Context, pool *Pool) error
	GetPool(ctx context.Context, id string) (*Pool, error)
	GetPoolByName(ctx context.Context, name string) (*Pool, error)
	UpdatePool(ctx context.Context, pool *Pool) error
	DeletePool(ctx context.Context, id string) error
	ListPools(ctx context.Context) ([]*Pool, error)
	ListPoolsByPurpose(ctx context.Context, purposeType string) ([]*Pool, error)

	AssignBackendToPool(ctx context.Context, poolID string, backendID string) error
	RemoveBackendFromPool(ctx context.Context, poolID string, backendID string) error
	ListBackendsForPool(ctx context.Context, poolID string) ([]*Backend, error)
	ListPoolsForBackend(ctx context.Context, backendID string) ([]*Pool, error)

	AssignModelToPool(ctx context.Context, poolID string, modelID string) error
	RemoveModelFromPool(ctx context.Context, poolID string, modelID string) error
	ListModelsForPool(ctx context.Context, poolID string) ([]*Model, error)
	ListPoolsForModel(ctx context.Context, modelID string) ([]*Pool, error)

	AppendJob(ctx context.Context, job Job) error
	AppendJobs(ctx context.Context, jobs ...*Job) error
	PopAllJobs(ctx context.Context) ([]*Job, error)
	PopJobsForType(ctx context.Context, taskType string) ([]*Job, error)
	PopNJobsForType(ctx context.Context, taskType string, n int) ([]*Job, error)
	PopJobForType(ctx context.Context, taskType string) (*Job, error)
	GetJobsForType(ctx context.Context, taskType string) ([]*Job, error)
	ListJobs(ctx context.Context, createdAtCursor *time.Time, limit int) ([]*Job, error)
	DeleteJobsByEntity(ctx context.Context, entityID, entityType string) error

	AppendLeasedJob(ctx context.Context, job Job, duration time.Duration, leaser string) error
	GetLeasedJob(ctx context.Context, id string) (*LeasedJob, error)
	DeleteLeasedJob(ctx context.Context, id string) error
	ListLeasedJobs(ctx context.Context, createdAtCursor *time.Time, limit int) ([]*LeasedJob, error)
	DeleteLeasedJobs(ctx context.Context, entityID, entityType string) error

	CreateAccessEntry(ctx context.Context, entry *AccessEntry) error
	GetAccessEntryByID(ctx context.Context, id string) (*AccessEntry, error)
	UpdateAccessEntry(ctx context.Context, entry *AccessEntry) error
	DeleteAccessEntry(ctx context.Context, id string) error
	DeleteAccessEntriesByIdentity(ctx context.Context, identity string) error
	DeleteAccessEntriesByResource(ctx context.Context, resource string) error
	ListAccessEntries(ctx context.Context, createdAtCursor time.Time) ([]*AccessEntry, error)
	GetAccessEntriesByIdentity(ctx context.Context, identity string) ([]*AccessEntry, error)
	GetAccessEntriesByIdentityAndResource(ctx context.Context, identity string, resource string) ([]*AccessEntry, error)

	CreateUser(ctx context.Context, user *User) error
	GetUserByID(ctx context.Context, id string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserBySubject(ctx context.Context, subject string) (*User, error)
	UpdateUser(ctx context.Context, user *User) error
	DeleteUser(ctx context.Context, id string) error
	ListUsersBySubjects(ctx context.Context, subject ...string) ([]*User, error)
	ListUsers(ctx context.Context, createdAtCursor time.Time) ([]*User, error)

	CreateFile(ctx context.Context, file *File) error
	GetFileByID(ctx context.Context, id string) (*File, error)
	UpdateFile(ctx context.Context, file *File) error
	DeleteFile(ctx context.Context, id string) error
	ListFiles(ctx context.Context) ([]string, error)

	EstimateFileCount(ctx context.Context) (int64, error)
	EnforceMaxFileCount(ctx context.Context, maxCount int64) error

	SetKV(ctx context.Context, key string, value json.RawMessage) error
	UpdateKV(ctx context.Context, key string, value json.RawMessage) error
	GetKV(ctx context.Context, key string, out interface{}) error
	DeleteKV(ctx context.Context, key string) error
	ListKV(ctx context.Context) ([]*KV, error)
	ListKVPrefix(ctx context.Context, prefix string) ([]*KV, error)

	ListFileIDsByParentID(ctx context.Context, parentID string) ([]string, error)
	CreateFileNameID(ctx context.Context, id, parentID, name string) error
	UpdateFileNameByID(ctx context.Context, id string, name string) error
	GetFileParentID(ctx context.Context, id string) (string, error)
	DeleteFileNameID(ctx context.Context, id string) error
	GetFileNameByID(ctx context.Context, id string) (string, error)
	ListFileIDsByName(ctx context.Context, parentID, name string) ([]string, error)
	UpdateFileParentID(ctx context.Context, id string, newParentID string) error

	CreateBlob(ctx context.Context, blob *Blob) error
	GetBlobByID(ctx context.Context, id string) (*Blob, error)
	DeleteBlob(ctx context.Context, id string) error

	AppendMessages(ctx context.Context, messages ...*Message) error
	DeleteMessages(ctx context.Context, stream string) error
	ListMessages(ctx context.Context, stream string) ([]*Message, error)
	LastMessage(ctx context.Context, stream string) (*Message, error)

	CreateMessageIndex(ctx context.Context, id string, identity string) error
	DeleteMessageIndex(ctx context.Context, id string, identity string) error
	ListMessageIndices(ctx context.Context, identity string) ([]string, error)

	CreateChunkIndex(ctx context.Context, chunk *ChunkIndex) error
	GetChunkIndexByID(ctx context.Context, id string) (*ChunkIndex, error)
	UpdateChunkIndex(ctx context.Context, chunk *ChunkIndex) error
	DeleteChunkIndex(ctx context.Context, id string) error
	ListChunkIndicesByVectorID(ctx context.Context, vectorID string) ([]*ChunkIndex, error)
	ListChunkIndicesByResource(ctx context.Context, resourceID, resourceType string) ([]*ChunkIndex, error)

	CreateGitHubRepo(ctx context.Context, repo *GitHubRepo) error
	GetGitHubRepo(ctx context.Context, id string) (*GitHubRepo, error)
	DeleteGitHubRepo(ctx context.Context, id string) error
	ListGitHubRepos(ctx context.Context) ([]*GitHubRepo, error)

	CreateTelegramFrontend(ctx context.Context, frontend *TelegramFrontend) error
	GetTelegramFrontend(ctx context.Context, id string) (*TelegramFrontend, error)
	UpdateTelegramFrontend(ctx context.Context, frontend *TelegramFrontend) error
	DeleteTelegramFrontend(ctx context.Context, id string) error
	ListTelegramFrontends(ctx context.Context) ([]*TelegramFrontend, error)
	ListTelegramFrontendsByUser(ctx context.Context, userID string) ([]*TelegramFrontend, error)

	CreateBot(ctx context.Context, bot *Bot) error
	GetBot(ctx context.Context, id string) (*Bot, error)
	GetBotByName(ctx context.Context, name string) (*Bot, error)
	UpdateBot(ctx context.Context, bot *Bot) error
	DeleteBot(ctx context.Context, id string) error
	ListBots(ctx context.Context) ([]*Bot, error)
	ListBotsByJobType(ctx context.Context, jobType string) ([]*Bot, error)
	ListBotsByUser(ctx context.Context, userID string) ([]*Bot, error)
}

//go:embed schema.sql
var Schema string

type store struct {
	libdb.Exec
}

func New(exec libdb.Exec) Store {
	if exec == nil {
		panic("SERVER BUG: store.New called with nil exec")
	}
	return &store{exec}
}

func quiet() func() {
	null, _ := os.Open(os.DevNull)
	sout := os.Stdout
	serr := os.Stderr
	os.Stdout = null
	os.Stderr = null
	log.SetOutput(null)
	return func() {
		defer null.Close()
		os.Stdout = sout
		os.Stderr = serr
		log.SetOutput(os.Stderr)
	}
}

// setupStore initializes a test Postgres instance and returns the store.
func SetupStore(t *testing.T) (context.Context, Store) {
	t.Helper()

	// Silence logs
	unquiet := quiet()
	t.Cleanup(unquiet)

	ctx := context.TODO()
	connStr, _, cleanup, err := libdb.SetupLocalInstance(ctx, "test", "test", "test")
	require.NoError(t, err)

	dbManager, err := libdb.NewPostgresDBManager(ctx, connStr, Schema)
	require.NoError(t, err)

	// Cleanup DB and container
	t.Cleanup(func() {
		require.NoError(t, dbManager.Close())
		cleanup()
	})

	s := New(dbManager.WithoutTransaction())
	return ctx, s
}

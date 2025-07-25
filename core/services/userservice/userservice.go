package userservice

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"dario.cat/mergo"
	"github.com/contenox/runtime-mvp/core/serverops"
	"github.com/contenox/runtime-mvp/core/serverops/store"
	"github.com/contenox/runtime-mvp/libs/libdb"
	"github.com/google/uuid"
)

var (
	ErrUserAlreadyExists     = errors.New("user already exists")
	ErrTokenGenerationFailed = errors.New("failed to generate token")
)

type Service interface {
	GetUserFromContext(ctx context.Context) (*store.User, error)
	Login(ctx context.Context, email, password string) (*Result, error)
	Register(ctx context.Context, req CreateUserRequest) (*Result, error)
	CreateUser(ctx context.Context, req CreateUserRequest) (*store.User, error)
	DeleteUser(ctx context.Context, id string) error
	UpdateUserFields(ctx context.Context, id string, req UpdateUserRequest) (*store.User, error)
	ListUsers(ctx context.Context, cursorCreatedAt time.Time) ([]*store.User, error)
	GetUserByID(ctx context.Context, id string) (*store.User, error)

	serverops.ServiceMeta
}

type service struct {
	dbInstance      libdb.DBManager
	securityEnabled bool
	serverSecret    string
	signingKey      string
}

func New(db libdb.DBManager, config *serverops.Config) Service {
	var securityEnabledFlag bool
	if config.SecurityEnabled == "true" {
		securityEnabledFlag = true
	}

	return &service{
		dbInstance:      db,
		securityEnabled: securityEnabledFlag,
		serverSecret:    config.JWTSecret,
		signingKey:      config.SigningKey,
	}
}

func (s *service) GetUserFromContext(ctx context.Context) (*store.User, error) {
	identity, err := serverops.GetIdentity(ctx)
	if err != nil {
		return nil, err
	}

	// Retrieve user by ID.
	user, err := s.getUserBySubject(ctx, identity)
	if err != nil {
		return nil, err
	}

	user.HashedPassword = ""
	user.RecoveryCodeHash = ""

	return user, nil
}

// Login authenticates a user given an email and password, and returns a JWT on success.
// It verifies the password, loads permissions, and generates a JWT token.
func (s *service) Login(ctx context.Context, email, password string) (*Result, error) {
	tx := s.dbInstance.WithoutTransaction()

	// Retrieve user by email.
	user, err := s.getUserByEmail(ctx, tx, email)
	if err != nil {
		return nil, err
	}
	if user.HashedPassword == "" {
		return nil, errors.New("direct login for this user is disabled")
	}
	passed, err := serverops.CheckPassword(password, user.HashedPassword, user.Salt, s.signingKey)
	if err != nil || !passed {
		return nil, errors.New("invalid credentials")
	}

	// Load permissions for the user.
	permissions, err := store.New(tx).GetAccessEntriesByIdentity(ctx, user.Subject)
	if err != nil {
		return nil, fmt.Errorf("failed to load permissions: %w", err)
	}

	// Use the serverops helper to generate the JWT.
	token, expiresAt, err := serverops.CreateAuthToken(user.Subject, permissions)
	if err != nil {
		return nil, err
	}
	user.HashedPassword = ""
	return &Result{User: user, Token: token, ExpiresAt: expiresAt}, nil
}

// Result bundles the newly registered user and its token.
type Result struct {
	User      *store.User `json:"user"`
	Token     string      `json:"token"`
	ExpiresAt time.Time   `json:"expires_at"`
}

// Register creates a new user and returns a JWT token for that user.
func (s *service) Register(ctx context.Context, req CreateUserRequest) (*Result, error) {
	tx := s.dbInstance.WithoutTransaction()
	req.AllowedResources = []CreateUserRequestAllowedResources{
		{Name: serverops.DefaultServerGroup, Permission: store.PermissionNone.String(), ResourceType: store.ResourceTypeSystem},
	}
	if serverops.DefaultAdminUser == req.Email {
		req.AllowedResources = []CreateUserRequestAllowedResources{
			{Name: serverops.DefaultServerGroup, Permission: store.PermissionManage.String(), ResourceType: store.ResourceTypeSystem},
		}
	}
	userFromStore, err := s.createUser(ctx, tx, req)
	if err != nil && !errors.Is(err, libdb.ErrNotFound) {
		return nil, fmt.Errorf("%w %w", ErrUserAlreadyExists, err)
	}
	if err != nil {
		return nil, err
	}

	permissions, err := store.New(tx).GetAccessEntriesByIdentity(ctx, userFromStore.Subject)
	if err != nil {
		return nil, fmt.Errorf("failed to load permissions: %w", err)
	}

	// Use the serverops helper to generate the token.
	token, expiresAt, err := serverops.CreateAuthToken(userFromStore.Subject, permissions)
	if err != nil {
		return nil, err
	}
	if token == "" {
		return nil, ErrTokenGenerationFailed
	}
	userFromStore.HashedPassword = ""
	return &Result{User: userFromStore, Token: token, ExpiresAt: expiresAt}, nil
}

type CreateUserRequest struct {
	Email            string                              `json:"email"`
	FriendlyName     string                              `json:"friendlyName,omitempty"`
	Password         string                              `json:"password"`
	AllowedResources []CreateUserRequestAllowedResources `json:"allowedResources"`
}

type CreateUserRequestAllowedResources struct {
	Name         string `json:"name"`
	Permission   string `json:"permission"`
	ResourceType string `json:"resourceType"`
}

func (s *service) CreateUser(ctx context.Context, req CreateUserRequest) (*store.User, error) {
	tx := s.dbInstance.WithoutTransaction()
	if err := serverops.CheckServiceAuthorization(ctx, store.New(tx), s, store.PermissionManage); err != nil {
		return nil, err
	}
	user, err := s.createUser(ctx, tx, req)
	if err != nil {
		return nil, err
	}
	user.HashedPassword = ""
	return user, nil
}

func (s *service) createUser(ctx context.Context, tx libdb.Exec, req CreateUserRequest) (*store.User, error) {
	id := uuid.NewString()
	user := &store.User{
		ID:           id,
		Subject:      id,
		Email:        req.Email,
		FriendlyName: req.FriendlyName,
	}
	if req.Password != "" {
		hashedPassword, salt, err := serverops.NewPasswordHash(req.Password, s.signingKey)
		if err != nil {
			return nil, err
		}
		user.HashedPassword = hashedPassword
		user.Salt = salt // Ensure store.User has a Salt field
	}

	err := store.New(tx).CreateUser(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	user, err = store.New(tx).GetUserByID(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get created user by id: %w", err)
	}
	for _, curar := range req.AllowedResources {
		perm, err := store.PermissionFromString(curar.Permission)
		if err != nil {
			return nil, err
		}
		err = store.New(tx).CreateAccessEntry(ctx, &store.AccessEntry{
			ID:           uuid.NewString(),
			Identity:     user.Subject,
			Resource:     curar.Name,
			ResourceType: curar.ResourceType,
			Permission:   perm,
		})
		if err != nil {
			return nil, err
		}
	}

	return user, nil
}

func (s *service) GetUserByID(ctx context.Context, id string) (*store.User, error) {
	tx := s.dbInstance.WithoutTransaction()
	if err := serverops.CheckServiceAuthorization(ctx, store.New(tx), s, store.PermissionManage); err != nil {
		return nil, err
	}

	return s.getUserByID(ctx, tx, id)
}

func (s *service) getUserByID(ctx context.Context, tx libdb.Exec, id string) (*store.User, error) {
	user, err := store.New(tx).GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return user, err
}

func (s *service) getUserByEmail(ctx context.Context, tx libdb.Exec, email string) (*store.User, error) {
	user, err := store.New(tx).GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *service) GetUserBySubject(ctx context.Context, subject string) (*store.User, error) {
	tx := s.dbInstance.WithoutTransaction()
	if err := serverops.CheckServiceAuthorization(ctx, store.New(tx), s, store.PermissionManage); err != nil {
		return nil, err
	}
	return s.getUserBySubject(ctx, subject)
}

func (s *service) getUserBySubject(ctx context.Context, subject string) (*store.User, error) {
	tx := s.dbInstance.WithoutTransaction()
	user, err := store.New(tx).GetUserBySubject(ctx, subject)
	if err != nil {
		return nil, err
	}
	return user, nil
}

type UpdateUserRequest struct {
	Email        string `json:"email,omitempty"`
	FriendlyName string `json:"friendlyName,omitempty"`
	Password     string `json:"password"`
}

// UpdateUserFields fetches the user, applies allowed updates, and persists the changes.
func (s *service) UpdateUserFields(ctx context.Context, id string, req UpdateUserRequest) (*store.User, error) {
	tx, commit, rTx, err := s.dbInstance.WithTransaction(ctx)
	defer func() {
		if err := rTx(); err != nil {
			log.Println("failed to rollback transaction", err)
		}
	}()
	if err != nil {
		return nil, err
	}
	if err := serverops.CheckServiceAuthorization(ctx, store.New(tx), s, store.PermissionManage); err != nil {
		return nil, err
	}
	// Retrieve the existing user
	user, err := s.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update allowed fields
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.FriendlyName != "" {
		user.FriendlyName = req.FriendlyName
	}
	if req.Password != "" {
		hashedPassword, salt, err := serverops.NewPasswordHash(req.Password, s.signingKey)
		if err != nil {
			return nil, err
		}
		user.HashedPassword = hashedPassword
		user.Salt = salt
	}

	// Persist the updated user. This method already handles merge logic and duplicate checks.
	if err := s.updateUser(ctx, tx, user); err != nil {
		return nil, err
	}

	if err := commit(ctx); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *service) updateUser(ctx context.Context, tx libdb.Exec, user *store.User) error {
	userDst, err := store.New(tx).GetUserByID(ctx, user.ID)
	if err != nil {
		return err
	}
	user.CreatedAt = userDst.CreatedAt
	user.RecoveryCodeHash = userDst.RecoveryCodeHash
	user.Subject = userDst.Subject
	err = mergo.Merge(userDst, user, mergo.WithOverride)
	if err != nil {
		return err
	}
	err = store.New(tx).UpdateUser(ctx, userDst)
	if err != nil {
		return err
	}
	return nil
}

func (s *service) DeleteUser(ctx context.Context, id string) error {
	tx, commit, rTx, err := s.dbInstance.WithTransaction(ctx)
	defer func() {
		if err := rTx(); err != nil {
			log.Println("failed to rollback transaction", err)
		}
	}()
	if err != nil {
		return err
	}
	if err := serverops.CheckServiceAuthorization(ctx, store.New(tx), s, store.PermissionManage); err != nil {
		return err
	}
	err = store.New(tx).DeleteUser(ctx, id)
	if err != nil {
		return err
	}
	err = store.New(tx).DeleteAccessEntriesByIdentity(ctx, id)
	if err != nil {
		return err
	}
	return commit(ctx)
}

func (s *service) ListUsers(ctx context.Context, cursorCreatedAt time.Time) ([]*store.User, error) {
	tx := s.dbInstance.WithoutTransaction()
	if err := serverops.CheckServiceAuthorization(ctx, store.New(tx), s, store.PermissionManage); err != nil {
		return nil, err
	}
	return store.New(tx).ListUsers(ctx, cursorCreatedAt)
}

func (s *service) GetServiceName() string {
	return "userservice"
}

func (s *service) GetServiceGroup() string {
	return serverops.DefaultDefaultServiceGroup
}

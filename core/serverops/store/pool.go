package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/contenox/runtime-mvp/libs/libdb"
)

func (s *store) CreatePool(ctx context.Context, pool *Pool) error {
	now := time.Now().UTC()
	pool.CreatedAt = now
	pool.UpdatedAt = now

	_, err := s.Exec.ExecContext(ctx, `
		INSERT INTO llm_pool
		(id, name, purpose_type, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)`,
		pool.ID, pool.Name, pool.PurposeType, pool.CreatedAt, pool.UpdatedAt,
	)
	return err
}

func (s *store) GetPool(ctx context.Context, id string) (*Pool, error) {
	var pool Pool
	err := s.Exec.QueryRowContext(ctx, `
		SELECT id, name, purpose_type, created_at, updated_at
		FROM llm_pool WHERE id = $1`, id,
	).Scan(&pool.ID, &pool.Name, &pool.PurposeType, &pool.CreatedAt, &pool.UpdatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, libdb.ErrNotFound
	}
	return &pool, err
}

func (s *store) GetPoolByName(ctx context.Context, name string) (*Pool, error) {
	var pool Pool
	err := s.Exec.QueryRowContext(ctx, `
		SELECT id, name, purpose_type, created_at, updated_at
		FROM llm_pool WHERE name = $1`, name,
	).Scan(&pool.ID, &pool.Name, &pool.PurposeType, &pool.CreatedAt, &pool.UpdatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, libdb.ErrNotFound
	}
	return &pool, err
}

func (s *store) UpdatePool(ctx context.Context, pool *Pool) error {
	pool.UpdatedAt = time.Now().UTC()

	result, err := s.Exec.ExecContext(ctx, `
		UPDATE llm_pool SET
		name = $2, purpose_type = $3, updated_at = $4
		WHERE id = $1`,
		pool.ID, pool.Name, pool.PurposeType, pool.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update pool: %w", err)
	}
	return checkRowsAffected(result)
}

func (s *store) DeletePool(ctx context.Context, id string) error {
	result, err := s.Exec.ExecContext(ctx, `
		DELETE FROM llm_pool WHERE id = $1`, id,
	)
	if err != nil {
		return fmt.Errorf("failed to delete pool: %w", err)
	}
	return checkRowsAffected(result)
}

func (s *store) ListPools(ctx context.Context) ([]*Pool, error) {
	rows, err := s.Exec.QueryContext(ctx, `
		SELECT id, name, purpose_type, created_at, updated_at
		FROM llm_pool ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pools []*Pool
	for rows.Next() {
		var pool Pool
		if err := rows.Scan(&pool.ID, &pool.Name, &pool.PurposeType, &pool.CreatedAt, &pool.UpdatedAt); err != nil {
			return nil, err
		}
		pools = append(pools, &pool)
	}
	return pools, rows.Err()
}

func (s *store) ListPoolsByPurpose(ctx context.Context, purposeType string) ([]*Pool, error) {
	rows, err := s.Exec.QueryContext(ctx, `
		SELECT id, name, purpose_type, created_at, updated_at
		FROM llm_pool WHERE purpose_type = $1
		ORDER BY created_at DESC`, purposeType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pools []*Pool
	for rows.Next() {
		var pool Pool
		if err := rows.Scan(&pool.ID, &pool.Name, &pool.PurposeType, &pool.CreatedAt, &pool.UpdatedAt); err != nil {
			return nil, err
		}
		pools = append(pools, &pool)
	}
	return pools, rows.Err()
}

func (s *store) AssignBackendToPool(ctx context.Context, poolID, backendID string) error {
	_, err := s.Exec.ExecContext(ctx, `
		INSERT INTO llm_pool_backend_assignments
		(pool_id, backend_id, assigned_at)
		VALUES ($1, $2, $3)`,
		poolID, backendID, time.Now().UTC())
	return err
}

func (s *store) RemoveBackendFromPool(ctx context.Context, poolID, backendID string) error {
	result, err := s.Exec.ExecContext(ctx, `
		DELETE FROM llm_pool_backend_assignments
		WHERE pool_id = $1 AND backend_id = $2`, poolID, backendID)
	if err != nil {
		return err
	}
	return checkRowsAffected(result)
}

func (s *store) ListBackendsForPool(ctx context.Context, poolID string) ([]*Backend, error) {
	rows, err := s.Exec.QueryContext(ctx, `
		SELECT b.id, b.name, b.base_url, b.type, b.created_at, b.updated_at
		FROM llm_backends b
		INNER JOIN llm_pool_backend_assignments a ON b.id = a.backend_id
		WHERE a.pool_id = $1
		ORDER BY a.assigned_at DESC`, poolID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var backends []*Backend
	for rows.Next() {
		var b Backend
		if err := rows.Scan(&b.ID, &b.Name, &b.BaseURL, &b.Type, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		backends = append(backends, &b)
	}
	return backends, rows.Err()
}

func (s *store) ListPoolsForBackend(ctx context.Context, backendID string) ([]*Pool, error) {
	rows, err := s.Exec.QueryContext(ctx, `
		SELECT p.id, p.name, p.purpose_type, p.created_at, p.updated_at
		FROM llm_pool p
		INNER JOIN llm_pool_backend_assignments a ON p.id = a.pool_id
		WHERE a.backend_id = $1
		ORDER BY a.assigned_at DESC`, backendID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pools []*Pool
	for rows.Next() {
		var p Pool
		if err := rows.Scan(&p.ID, &p.Name, &p.PurposeType, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		pools = append(pools, &p)
	}
	return pools, rows.Err()
}

func (s *store) AssignModelToPool(ctx context.Context, poolID, modelID string) error {
	now := time.Now().UTC()
	_, err := s.Exec.ExecContext(ctx, `
		INSERT INTO ollama_model_assignments
		(model_id, llm_pool_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4)`, modelID, poolID, now, now)
	return err
}

func (s *store) RemoveModelFromPool(ctx context.Context, poolID, modelID string) error {
	result, err := s.Exec.ExecContext(ctx, `
		DELETE FROM ollama_model_assignments
		WHERE model_id = $1 AND llm_pool_id = $2`, modelID, poolID)
	if err != nil {
		return err
	}
	return checkRowsAffected(result)
}

func (s *store) ListModelsForPool(ctx context.Context, poolID string) ([]*Model, error) {
	rows, err := s.Exec.QueryContext(ctx, `
		SELECT m.id, m.model, m.created_at, m.updated_at
		FROM ollama_models m
		INNER JOIN ollama_model_assignments a ON m.id = a.model_id
		WHERE a.llm_pool_id = $1
		ORDER BY a.created_at DESC`, poolID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var models []*Model
	for rows.Next() {
		var m Model
		if err := rows.Scan(&m.ID, &m.Model, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		models = append(models, &m)
	}
	return models, rows.Err()
}

func (s *store) ListPoolsForModel(ctx context.Context, modelID string) ([]*Pool, error) {
	rows, err := s.Exec.QueryContext(ctx, `
		SELECT p.id, p.name, p.purpose_type, p.created_at, p.updated_at
		FROM llm_pool p
		INNER JOIN ollama_model_assignments a ON p.id = a.llm_pool_id
		WHERE a.model_id = $1
		ORDER BY a.created_at DESC`, modelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pools []*Pool
	for rows.Next() {
		var p Pool
		if err := rows.Scan(&p.ID, &p.Name, &p.PurposeType, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		pools = append(pools, &p)
	}
	return pools, rows.Err()
}

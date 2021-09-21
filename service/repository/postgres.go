package repository

import (
	"context"
	"fmt"
	"github.com/beldeveloper/app-lego/model"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

// NewPostgres creates a new instance of the repositories service.
func NewPostgres(conn *pgxpool.Pool, schema string) Postgres {
	return Postgres{conn: conn, schema: schema}
}

// Postgres implements the repositories service with the Postgres storage.
type Postgres struct {
	conn   *pgxpool.Pool
	schema string
}

// FindByID returns a repository by its ID.
func (p Postgres) FindByID(ctx context.Context, id uint64) (model.Repository, error) {
	var r model.Repository
	q := fmt.Sprintf(
		`SELECT "id", "type", "alias", "name", "status", "updated_at" FROM "%s"."repositories" WHERE "id" = $1`,
		p.schema,
	)
	err := p.conn.QueryRow(ctx, q, id).Scan(&r.ID, &r.Type, &r.Alias, &r.Name, &r.Status, &r.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return r, model.ErrNotFound
		}
		return r, fmt.Errorf("service.repository.postgres.FindByID: scan: %w; id = %d", err, id)
	}
	return r, nil
}

// FindPending returns a repository that is awaiting to be downloaded,
func (p Postgres) FindPending(ctx context.Context) (model.Repository, error) {
	var r model.Repository
	q := fmt.Sprintf(
		`SELECT "id", "type", "alias", "name", "status", "updated_at"
		FROM "%s"."repositories" WHERE "status" = $1 LIMIT 1`,
		p.schema,
	)
	err := p.conn.QueryRow(ctx, q, model.RepositoryStatusPending).
		Scan(&r.ID, &r.Type, &r.Alias, &r.Name, &r.Status, &r.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return r, model.ErrNotFound
		}
		return r, fmt.Errorf("service.repository.postgres.FindPending: scan: %w", err)
	}
	return r, nil
}

// FindOutdated return the repository that is ready for pulling updates longer that others.
func (p Postgres) FindOutdated(ctx context.Context) (model.Repository, error) {
	var r model.Repository
	q := fmt.Sprintf(
		`SELECT "id", "type", "alias", "name", "status", "updated_at"
		FROM "%s"."repositories" WHERE "status" = $1 ORDER BY "updated_at" ASC LIMIT 1`,
		p.schema,
	)
	err := p.conn.QueryRow(ctx, q, model.RepositoryStatusReady).
		Scan(&r.ID, &r.Type, &r.Alias, &r.Name, &r.Status, &r.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return r, model.ErrNotFound
		}
		return r, fmt.Errorf("service.repository.postgres.FindOutdated: scan: %w", err)
	}
	return r, nil
}

// Add saves a new repository.
func (p Postgres) Add(ctx context.Context, r model.Repository) (model.Repository, error) {
	q := fmt.Sprintf(
		`INSERT INTO "%s"."repositories" ("type", "alias", "name", "status", "updated_at")
		VALUES ($1, $2, $3, $4, $5) RETURNING "id"`,
		p.schema,
	)
	err := p.conn.QueryRow(ctx, q, r.Type, r.Alias, r.Name, r.Status, r.UpdatedAt).Scan(&r.ID)
	if err != nil {
		return r, fmt.Errorf("service.repository.postgres.Add: insert: %w", err)
	}
	return r, nil
}

// Update modifies a specific repository.
func (p Postgres) Update(ctx context.Context, r model.Repository) (model.Repository, error) {
	q := fmt.Sprintf(`UPDATE "%s"."repositories" SET "updated_at" = $2, "status" = $3 WHERE "id" = $1`, p.schema)
	_, err := p.conn.Exec(ctx, q, r.ID, r.UpdatedAt, r.Status)
	if err != nil {
		return r, fmt.Errorf("service.repository.postgres.Update: exec: %w", err)
	}
	return r, nil
}

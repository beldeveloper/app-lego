package repository

import (
	"context"
	"fmt"
	"github.com/beldeveloper/app-lego/model"
	"github.com/beldeveloper/go-errors-context"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

// NewPostgres creates a new instance of the repositories service.
func NewPostgres(conn *pgxpool.Pool, schema model.PgSchema) Service {
	return Postgres{conn: conn, schema: string(schema)}
}

// Postgres implements the repositories service with the Postgres storage.
type Postgres struct {
	conn   *pgxpool.Pool
	schema string
}

// FindAll returns all repositories.
func (p Postgres) FindAll(ctx context.Context) ([]model.Repository, error) {
	q := fmt.Sprintf(
		`SELECT "id", "type", "alias", "name", "status", "cfg_file", "updated_at"
		FROM "%s"."repositories" ORDER BY "alias"`,
		p.schema,
	)
	rows, err := p.conn.Query(ctx, q)
	if err != nil {
		return nil, errors.WrapContext(err, errors.Context{Path: "service.repository.postgres.FindAll: query"})
	}
	defer rows.Close()
	res := make([]model.Repository, 0)
	var r model.Repository
	for rows.Next() {
		err = rows.Scan(&r.ID, &r.Type, &r.Alias, &r.Name, &r.Status, &r.CfgFile, &r.UpdatedAt)
		if err != nil {
			return nil, errors.WrapContext(err, errors.Context{Path: "service.repository.postgres.FindAll: scan"})
		}
		res = append(res, r)
	}
	return res, nil
}

// FindByID returns a repository by its ID.
func (p Postgres) FindByID(ctx context.Context, id uint64) (model.Repository, error) {
	var r model.Repository
	q := fmt.Sprintf(
		`SELECT "id", "type", "alias", "name", "status", "cfg_file", "updated_at"
		FROM "%s"."repositories" WHERE "id" = $1`,
		p.schema,
	)
	err := p.conn.QueryRow(ctx, q, id).Scan(&r.ID, &r.Type, &r.Alias, &r.Name, &r.Status, &r.CfgFile, &r.UpdatedAt)
	if err == pgx.ErrNoRows {
		err = model.ErrNotFound
	}
	return r, errors.WrapContext(err, errors.Context{
		Path:   "service.repository.postgres.FindByID: scan",
		Params: errors.Params{"repository": id},
	})
}

// FindPending returns a repository that is awaiting to be downloaded,
func (p Postgres) FindPending(ctx context.Context) (model.Repository, error) {
	var r model.Repository
	q := fmt.Sprintf(
		`SELECT "id", "type", "alias", "name", "status", "cfg_file", "updated_at"
		FROM "%s"."repositories" WHERE "status" = $1 LIMIT 1`,
		p.schema,
	)
	err := p.conn.QueryRow(ctx, q, model.RepositoryStatusPending).
		Scan(&r.ID, &r.Type, &r.Alias, &r.Name, &r.Status, &r.CfgFile, &r.UpdatedAt)
	if err == pgx.ErrNoRows {
		return r, model.ErrNotFound
	}
	return r, errors.WrapContext(err, errors.Context{Path: "service.repository.postgres.FindPending: scan"})
}

// FindOutdated return the repository that is ready for pulling updates longer that others.
func (p Postgres) FindOutdated(ctx context.Context) (model.Repository, error) {
	var r model.Repository
	q := fmt.Sprintf(
		`SELECT "id", "type", "alias", "name", "status", "cfg_file", "updated_at"
		FROM "%s"."repositories" WHERE "status" = $1 ORDER BY "updated_at" ASC LIMIT 1`,
		p.schema,
	)
	err := p.conn.QueryRow(ctx, q, model.RepositoryStatusReady).
		Scan(&r.ID, &r.Type, &r.Alias, &r.Name, &r.Status, &r.CfgFile, &r.UpdatedAt)
	if err == pgx.ErrNoRows {
		return r, model.ErrNotFound
	}
	return r, errors.WrapContext(err, errors.Context{Path: "service.repository.postgres.FindOutdated: scan"})
}

// Add saves a new repository.
func (p Postgres) Add(ctx context.Context, r model.Repository) (model.Repository, error) {
	q := fmt.Sprintf(
		`INSERT INTO "%s"."repositories" ("type", "alias", "name", "status", "cfg_file", "updated_at")
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING "id"`,
		p.schema,
	)
	err := p.conn.QueryRow(ctx, q, r.Type, r.Alias, r.Name, r.Status, r.CfgFile, r.UpdatedAt).Scan(&r.ID)
	return r, errors.WrapContext(err, errors.Context{Path: "service.repository.postgres.Add: scan"})
}

// Update modifies a specific repository.
func (p Postgres) Update(ctx context.Context, r model.Repository) (model.Repository, error) {
	q := fmt.Sprintf(
		`UPDATE "%s"."repositories" SET "updated_at" = $2, "status" = $3, "cfg_file" = $4 WHERE "id" = $1`,
		p.schema,
	)
	_, err := p.conn.Exec(ctx, q, r.ID, r.UpdatedAt, r.Status, r.CfgFile)
	return r, errors.WrapContext(err, errors.Context{
		Path:   "service.repository.postgres.Update: exec",
		Params: errors.Params{"repository": r.ID, "status": r.Status},
	})
}

// LoadSecrets reads the secret variables for the specific repository.
func (p Postgres) LoadSecrets(ctx context.Context, r model.Repository) ([]model.Variable, error) {
	var res []model.Variable
	q := fmt.Sprintf(`SELECT "secrets" FROM "%s"."repositories" WHERE "id" = $1`, p.schema)
	err := p.conn.QueryRow(ctx, q, r.ID).Scan(&res)
	if err != nil {
		if err == pgx.ErrNoRows {
			err = model.ErrNotFound
		}
		return nil, errors.WrapContext(err, errors.Context{
			Path:   "service.repository.postgres.LoadSecrets: scan",
			Params: errors.Params{"repository": r.ID},
		})
	}
	for i := range res {
		res[i].Type = model.VariableTypeSecret
	}
	return res, nil
}

// SaveSecrets saves the secret variables for the specific repository.
func (p Postgres) SaveSecrets(ctx context.Context, r model.Repository, secrets []model.Variable) error {
	q := fmt.Sprintf(`UPDATE "%s"."repositories" SET "secrets" = $2 WHERE "id" = $1`, p.schema)
	_, err := p.conn.Exec(ctx, q, r.ID, secrets)
	if err != nil {
		return errors.WrapContext(err, errors.Context{
			Path:   "service.repository.postgres.SaveSecrets: exec",
			Params: errors.Params{"repository": r.ID},
		})
	}
	return nil
}

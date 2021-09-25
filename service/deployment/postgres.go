package deployment

import (
	"context"
	"fmt"
	"github.com/beldeveloper/app-lego/model"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

// NewPostgres creates a new instance of the deployments service.
func NewPostgres(conn *pgxpool.Pool, schema string) Postgres {
	return Postgres{conn: conn, schema: schema}
}

// Postgres implements the deployments service with the Postgres storage.
type Postgres struct {
	conn   *pgxpool.Pool
	schema string
}

// FindAll returns all deployments.
func (p Postgres) FindAll(ctx context.Context) ([]model.Deployment, error) {
	q := fmt.Sprintf(
		`SELECT "id", "status", "created_at", "auto_rebuild", "branches" FROM "%s"."deployments" ORDER BY "created_at" DESC`,
		p.schema,
	)
	rows, err := p.conn.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("service.deployment.postgres.FindAll: query: %w", err)
	}
	defer rows.Close()
	res := make([]model.Deployment, 0)
	var d model.Deployment
	for rows.Next() {
		d.Branches = nil
		err = rows.Scan(&d.ID, &d.Status, &d.CreatedAt, &d.AutoRebuild, &d.Branches)
		if err != nil {
			return nil, fmt.Errorf("service.deployment.postgres.FindAll: scan: %w", err)
		}
		res = append(res, d)
	}
	return res, nil
}

// FindForAutoRebuild returns all ready deployments that are marked as auto_rebuild and are bound to the specific branch.
func (p Postgres) FindForAutoRebuild(ctx context.Context, b model.Branch) ([]model.Deployment, error) {
	q := fmt.Sprintf(
		`SELECT "d"."id", "d"."status", "d"."created_at", "d"."auto_rebuild", "d"."branches"
		FROM "%s"."deployments" "d"
		CROSS JOIN LATERAL JSONB_ARRAY_ELEMENTS("d"."branches") AS "b"
		WHERE ("b"->>'id')::int = $1 AND "d"."auto_rebuild" = TRUE AND "d"."status" = $2
		ORDER BY "created_at" DESC`,
		p.schema,
	)
	rows, err := p.conn.Query(ctx, q, b.ID, model.DeploymentStatusReady)
	if err != nil {
		return nil, fmt.Errorf("service.deployment.postgres.FindForAutoRebuild: query: %w", err)
	}
	defer rows.Close()
	res := make([]model.Deployment, 0)
	var d model.Deployment
	for rows.Next() {
		d.Branches = nil
		err = rows.Scan(&d.ID, &d.Status, &d.CreatedAt, &d.AutoRebuild, &d.Branches)
		if err != nil {
			return nil, fmt.Errorf("service.deployment.postgres.FindForAutoRebuild: scan: %w", err)
		}
		res = append(res, d)
	}
	return res, nil
}

// FindByID returns the one deployment with the specific ID.
func (p Postgres) FindByID(ctx context.Context, id uint64) (model.Deployment, error) {
	var d model.Deployment
	q := fmt.Sprintf(
		`SELECT "id", "status", "created_at", "auto_rebuild", "branches" FROM "%s"."deployments" WHERE "id" = $1`,
		p.schema,
	)
	err := p.conn.QueryRow(ctx, q, id).Scan(&d.ID, &d.Status, &d.CreatedAt, &d.AutoRebuild, &d.Branches)
	if err != nil {
		if err == pgx.ErrNoRows {
			return d, model.ErrNotFound
		}
		return d, fmt.Errorf("service.deployment.postgres.FindByID: query: %w", err)
	}
	return d, nil
}

// Add saves a new deployment.
func (p Postgres) Add(ctx context.Context, d model.Deployment) (model.Deployment, error) {
	q := fmt.Sprintf(
		`INSERT INTO "%s"."deployments" ("status", "created_at", "auto_rebuild", "branches")
		VALUES ($1, $2, $3, $4) RETURNING "id"`,
		p.schema,
	)
	err := p.conn.QueryRow(ctx, q, d.Status, d.CreatedAt, d.AutoRebuild, d.Branches).Scan(&d.ID)
	if err != nil {
		return d, fmt.Errorf("service.deployment.postgres.Add: insert: %w", err)
	}
	return d, nil
}

// Update modifies a specific deployment.
func (p Postgres) Update(ctx context.Context, d model.Deployment) (model.Deployment, error) {
	q := fmt.Sprintf(`UPDATE "%s"."deployments" SET "status" = $2, "branches" = $3 WHERE "id" = $1`, p.schema)
	_, err := p.conn.Exec(ctx, q, d.ID, d.Status, d.Branches)
	if err != nil {
		return d, fmt.Errorf("service.deployment.postgres.Update: exec: %w", err)
	}
	return d, nil
}

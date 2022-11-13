package postgres

import (
	"context"
	"github.com/beldeveloper/app-lego/internal/app"
	"github.com/beldeveloper/app-lego/internal/app/errtype"
	"github.com/beldeveloper/go-errors-context"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

// NewDeployment creates a new instance of the repository.
func NewDeployment(conn *pgxpool.Pool) app.DeploymentRepo {
	return Deployment{conn: conn}
}

// Deployment implements a repository.
type Deployment struct {
	conn *pgxpool.Pool
}

// FindAll returns non-closed deployments.
func (r Deployment) FindAll(ctx context.Context) ([]app.Deployment, error) {
	q := `SELECT "id", "status", "created_at", "auto_rebuild", "branches" FROM "deployments"
		WHERE "status" != $1 ORDER BY "created_at" DESC`
	rows, err := r.conn.Query(ctx, q, app.DeploymentStatusClosed)
	if err != nil {
		return nil, errors.WrapContext(err, errors.Context{Path: "postgres.Deployment.FindAll.Query"})
	}
	defer rows.Close()
	res := make([]app.Deployment, 0)
	var d app.Deployment
	for rows.Next() {
		d.Branches = nil
		err = rows.Scan(&d.ID, &d.Status, &d.CreatedAt, &d.AutoRebuild, &d.Branches)
		if err != nil {
			return nil, errors.WrapContext(err, errors.Context{Path: "postgres.Deployment.FindAll.Scan"})
		}
		res = append(res, d)
	}
	return res, nil
}

// FindForAutoRebuild returns all ready deployments that are marked as auto_rebuild and are bound to the specific branch.
func (r Deployment) FindForAutoRebuild(ctx context.Context, b app.Branch) ([]app.Deployment, error) {
	q := `SELECT "d"."id", "d"."status", "d"."created_at", "d"."auto_rebuild", "d"."branches"
		FROM "deployments" "d"
		CROSS JOIN LATERAL JSONB_ARRAY_ELEMENTS("d"."branches") AS "b"
		WHERE ("b"->>'id')::int = $1 AND "d"."auto_rebuild" = TRUE AND "d"."status" = $2
		ORDER BY "created_at" DESC`
	rows, err := r.conn.Query(ctx, q, b.ID, app.DeploymentStatusReady)
	if err != nil {
		return nil, errors.WrapContext(err, errors.Context{
			Path:   "postgres.Deployment.FindForAutoRebuild.Query",
			Params: errors.Params{"branch": b.ID},
		})
	}
	defer rows.Close()
	res := make([]app.Deployment, 0)
	var d app.Deployment
	for rows.Next() {
		d.Branches = nil
		err = rows.Scan(&d.ID, &d.Status, &d.CreatedAt, &d.AutoRebuild, &d.Branches)
		if err != nil {
			return nil, errors.WrapContext(err, errors.Context{
				Path:   "postgres.Deployment.FindForAutoRebuild.Scan",
				Params: errors.Params{"branch": b.ID},
			})
		}
		res = append(res, d)
	}
	return res, nil
}

// FindByID returns the one deployment with the specific ID.
func (r Deployment) FindByID(ctx context.Context, id uint64) (app.Deployment, error) {
	var d app.Deployment
	q := `SELECT "id", "status", "created_at", "auto_rebuild", "branches" FROM "deployments" WHERE "id" = $1`
	err := r.conn.QueryRow(ctx, q, id).Scan(&d.ID, &d.Status, &d.CreatedAt, &d.AutoRebuild, &d.Branches)
	if err == pgx.ErrNoRows {
		err = errtype.ErrNotFound
	}
	return d, errors.WrapContext(err, errors.Context{
		Path:   "postgres.Deployment.FindByID.Scan",
		Params: errors.Params{"deployment": id},
	})
}

// Add saves a new deployment.
func (r Deployment) Add(ctx context.Context, d app.Deployment) (app.Deployment, error) {
	q := `INSERT INTO "deployments" ("status", "created_at", "auto_rebuild", "branches")
		VALUES ($1, $2, $3, $4) RETURNING "id"`
	err := r.conn.QueryRow(ctx, q, d.Status, d.CreatedAt, d.AutoRebuild, d.Branches).Scan(&d.ID)
	return d, errors.WrapContext(err, errors.Context{Path: "postgres.Deployment.Add.Scan"})
}

// Update modifies a specific deployment.
func (r Deployment) Update(ctx context.Context, d app.Deployment) (app.Deployment, error) {
	q := `UPDATE "deployments" SET "status" = $2, "branches" = $3 WHERE "id" = $1`
	_, err := r.conn.Exec(ctx, q, d.ID, d.Status, d.Branches)
	return d, errors.WrapContext(err, errors.Context{
		Path:   "postgres.Deployment.Update.Exec",
		Params: errors.Params{"deployment": d.ID},
	})
}

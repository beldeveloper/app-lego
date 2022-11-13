package postgres

import (
	"context"
	"github.com/beldeveloper/app-lego/internal/app"
	"github.com/beldeveloper/app-lego/internal/app/errtype"
	"github.com/beldeveloper/go-errors-context"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

// NewRepository creates a new instance of the repository.
func NewRepository(conn *pgxpool.Pool) app.RepositoryRepo {
	return Repository{conn: conn}
}

// Repository (vcs) implements a (db) repository.
type Repository struct {
	conn *pgxpool.Pool
}

// FindAll repositories.
func (r Repository) FindAll(ctx context.Context) ([]app.Repository, error) {
	q := `SELECT "id", "type", "alias", "name", "status", "updated_at" FROM "repositories" ORDER BY "alias"`
	rows, err := r.conn.Query(ctx, q)
	if err != nil {
		return nil, errors.WrapContext(err, errors.Context{Path: "postgres.repository.FindAll.query"})
	}
	defer rows.Close()
	res := make([]app.Repository, 0, 30)
	var repo app.Repository
	for rows.Next() {
		err = rows.Scan(&repo.ID, &repo.Type, &repo.Alias, &repo.Name, &repo.Status, &repo.UpdatedAt)
		if err != nil {
			return nil, errors.WrapContext(err, errors.Context{Path: "postgres.repository.FindAll.scan"})
		}
		res = append(res, repo)
	}
	return res, nil
}

// FindByID returns a repository by its ID.
func (r Repository) FindByID(ctx context.Context, id uint64) (app.Repository, error) {
	var repo app.Repository
	q := `SELECT "id", "type", "alias", "name", "status", "updated_at" FROM "repositories" WHERE "id" = $1`
	err := r.conn.QueryRow(ctx, q, id).Scan(&repo.ID, &repo.Type, &repo.Alias, &repo.Name, &repo.Status, &repo.UpdatedAt)
	if err == pgx.ErrNoRows {
		err = errtype.ErrNotFound
	}
	return repo, errors.WrapContext(err, errors.Context{
		Path:   "postgres.repository.FindByID.scan",
		Params: errors.Params{"repository": id},
	})
}

// FindPending returns a repository that is awaiting to be downloaded,
func (r Repository) FindPending(ctx context.Context) (app.Repository, error) {
	var repo app.Repository
	q := `SELECT "id", "type", "alias", "name", "status", "updated_at" FROM "repositories" WHERE "status" = $1 LIMIT 1`
	err := r.conn.QueryRow(ctx, q, app.RepositoryStatusPending).
		Scan(&repo.ID, &repo.Type, &repo.Alias, &repo.Name, &repo.Status, &repo.UpdatedAt)
	if err == pgx.ErrNoRows {
		return repo, errtype.ErrNotFound
	}
	return repo, errors.WrapContext(err, errors.Context{Path: "postgres.repository.FindPending.scan"})
}

// FindOutdated return the repository that is ready for pulling updates longer that others.
func (r Repository) FindOutdated(ctx context.Context) (app.Repository, error) {
	var repo app.Repository
	q := `SELECT "id", "type", "alias", "name", "status", "updated_at"
		FROM "repositories" WHERE "status" = $1 ORDER BY "updated_at" ASC LIMIT 1`
	err := r.conn.QueryRow(ctx, q, app.RepositoryStatusReady).
		Scan(&repo.ID, &repo.Type, &repo.Alias, &repo.Name, &repo.Status, &repo.UpdatedAt)
	if err == pgx.ErrNoRows {
		return repo, errtype.ErrNotFound
	}
	return repo, errors.WrapContext(err, errors.Context{Path: "postgres.repository.FindOutdated.scan"})
}

// Add saves a new repository.
func (r Repository) Add(ctx context.Context, repo app.Repository) (app.Repository, error) {
	q := `INSERT INTO "repositories" ("type", "alias", "name", "status", "updated_at")
		VALUES ($1, $2, $3, $4, $5) RETURNING "id"`
	err := r.conn.QueryRow(ctx, q, repo.Type, repo.Alias, repo.Name, repo.Status, repo.UpdatedAt).Scan(&repo.ID)
	return repo, errors.WrapContext(err, errors.Context{Path: "postgres.repository.Add.scan"})
}

// Update modifies a specific repository.
func (r Repository) Update(ctx context.Context, repo app.Repository) (app.Repository, error) {
	q := `UPDATE "repositories" SET "updated_at" = $2, "status" = $3 WHERE "id" = $1`
	_, err := r.conn.Exec(ctx, q, repo.ID, repo.UpdatedAt, repo.Status)
	return repo, errors.WrapContext(err, errors.Context{
		Path:   "postgres.repository.Add.Exec",
		Params: errors.Params{"repository": repo.ID, "status": repo.Status},
	})
}

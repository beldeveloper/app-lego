package postgres

import (
	"context"
	"fmt"
	"github.com/beldeveloper/app-lego/internal/app"
	"github.com/beldeveloper/app-lego/internal/app/errtype"
	"github.com/beldeveloper/go-errors-context"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"strconv"
	"strings"
)

// NewBranch creates a new instance of the repository.
func NewBranch(conn *pgxpool.Pool) app.BranchRepo {
	return Branch{conn: conn}
}

// Branch implements a repository.
type Branch struct {
	conn *pgxpool.Pool
}

// FindAll returns all branches.
func (r Branch) FindAll(ctx context.Context) ([]app.Branch, error) {
	q := `SELECT "id", "repository_id", "type", "name", "hash", "status" FROM "branches" ORDER BY "name"`
	rows, err := r.conn.Query(ctx, q)
	if err != nil {
		return nil, errors.WrapContext(err, errors.Context{Path: "postgres.Branch.FindAll.Query"})
	}
	defer rows.Close()
	res := make([]app.Branch, 0)
	var b app.Branch
	for rows.Next() {
		err = rows.Scan(&b.ID, &b.RepositoryID, &b.Type, &b.Name, &b.Hash, &b.Status)
		if err != nil {
			return nil, errors.WrapContext(err, errors.Context{Path: "postgres.Branch.FindAll.Scan"})
		}
		res = append(res, b)
	}
	return res, nil
}

// FindByIDs returns all branches with the specific IDs.
func (r Branch) FindByIDs(ctx context.Context, ids []uint64) ([]app.Branch, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	idsStr := make([]string, len(ids))
	for i, id := range ids {
		idsStr[i] = strconv.Itoa(int(id))
	}
	q := fmt.Sprintf(
		`SELECT "id", "repository_id", "type", "name", "hash", "status" FROM "branches" WHERE "id" IN (%s)`,
		strings.Join(idsStr, ","),
	)
	rows, err := r.conn.Query(ctx, q)
	if err != nil {
		return nil, errors.WrapContext(err, errors.Context{
			Path:   "postgres.Branch.FindByIDs.Query",
			Params: errors.Params{"ids": ids},
		})
	}
	defer rows.Close()
	res := make([]app.Branch, 0)
	var b app.Branch
	for rows.Next() {
		err = rows.Scan(&b.ID, &b.RepositoryID, &b.Type, &b.Name, &b.Hash, &b.Status)
		if err != nil {
			return nil, errors.WrapContext(err, errors.Context{
				Path:   "postgres.Branch.FindByIDs.Scan",
				Params: errors.Params{"ids": ids},
			})
		}
		res = append(res, b)
	}
	return res, nil
}

// FindByRepository returns all branches that belong to the specific repository.
func (r Branch) FindByRepository(ctx context.Context, repo app.Repository) ([]app.Branch, error) {
	q := `SELECT "id", "repository_id", "type", "name", "hash", "status" FROM "branches" WHERE "repository_id" = $1`
	rows, err := r.conn.Query(ctx, q, repo.ID)
	if err != nil {
		return nil, errors.WrapContext(err, errors.Context{
			Path:   "postgres.Branch.FindByRepository.Query",
			Params: errors.Params{"repository": repo.ID},
		})
	}
	defer rows.Close()
	res := make([]app.Branch, 0)
	var b app.Branch
	for rows.Next() {
		err = rows.Scan(&b.ID, &b.RepositoryID, &b.Type, &b.Name, &b.Hash, &b.Status)
		if err != nil {
			return nil, errors.WrapContext(err, errors.Context{
				Path:   "postgres.Branch.FindByRepository.Scan",
				Params: errors.Params{"repository": repo.ID},
			})
		}
		res = append(res, b)
	}
	return res, nil
}

// FindByID returns the one branch with the specific ID.
func (r Branch) FindByID(ctx context.Context, id uint64) (app.Branch, error) {
	var b app.Branch
	q := `SELECT "id", "repository_id", "type", "name", "hash", "status" FROM "branches" WHERE "id" = $1`
	err := r.conn.QueryRow(ctx, q, id).Scan(&b.ID, &b.RepositoryID, &b.Type, &b.Name, &b.Hash, &b.Status)
	if err == pgx.ErrNoRows {
		err = errtype.ErrNotFound
	}
	return b, errors.WrapContext(err, errors.Context{
		Path:   "postgres.Branch.FindByID.Scan",
		Params: errors.Params{"branch": id},
	})
}

// FindEnqueued returns the one branch that is enqueued or in building status (it means the process was interrupted earlier).
func (r Branch) FindEnqueued(ctx context.Context) (app.Branch, error) {
	var b app.Branch
	q := `SELECT "id", "repository_id", "type", "name", "hash", "status" FROM "branches" WHERE "status" IN ($1, $2) LIMIT 1`
	err := r.conn.QueryRow(ctx, q, app.BranchStatusEnqueued, app.BranchStatusBuilding).
		Scan(&b.ID, &b.RepositoryID, &b.Type, &b.Name, &b.Hash, &b.Status)
	if err == pgx.ErrNoRows {
		err = errtype.ErrNotFound
	}
	return b, errors.WrapContext(err, errors.Context{Path: "postgres.Branch.FindEnqueued.Scan"})
}

// Add saves a new branch.
func (r Branch) Add(ctx context.Context, b app.Branch) (app.Branch, error) {
	q := `INSERT INTO "branches" ("repository_id", "type", "name", "hash", "status")
		VALUES ($1, $2, $3, $4, $5) RETURNING "id"`
	err := r.conn.QueryRow(ctx, q, b.RepositoryID, b.Type, b.Name, b.Hash, b.Status).Scan(&b.ID)
	if err != nil {
		return b, errors.WrapContext(err, errors.Context{Path: "postgres.Branch.Add.Scan"})
	}
	return b, nil
}

// Update modifies a specific branch.
func (r Branch) Update(ctx context.Context, b app.Branch) (app.Branch, error) {
	q := `UPDATE "branches" SET "hash" = $2, "status" = $3 WHERE "id" = $1`
	_, err := r.conn.Exec(ctx, q, b.ID, b.Hash, b.Status)
	return b, errors.WrapContext(err, errors.Context{
		Path:   "postgres.Branch.Update.Exec",
		Params: errors.Params{"branch": b.ID, "status": b.Status, "hash": b.Hash},
	})
}

// DeleteByIDs deletes all branches with the specific ids.
func (r Branch) DeleteByIDs(ctx context.Context, ids []uint64) error {
	if len(ids) == 0 {
		return nil
	}
	idsStr := make([]string, len(ids))
	for i, id := range ids {
		idsStr[i] = strconv.Itoa(int(id))
	}
	q := fmt.Sprintf(`DELETE FROM "branches" WHERE "id" IN (%s)`, strings.Join(idsStr, ","))
	_, err := r.conn.Exec(ctx, q)
	if err != nil {
		return errors.WrapContext(err, errors.Context{
			Path:   "postgres.Branch.DeleteByIDs.Exec",
			Params: errors.Params{"ids": ids},
		})
	}
	return nil
}

// UpdateStatus modifies the branch status.
func (r Branch) UpdateStatus(ctx context.Context, b app.Branch) error {
	q := `UPDATE "branches" SET "status" = $2 WHERE "id" = $1`
	_, err := r.conn.Exec(ctx, q, b.ID, b.Status)
	return errors.WrapContext(err, errors.Context{
		Path:   "postgres.Branch.UpdateStatus.Exec",
		Params: errors.Params{"branch": b.ID, "status": b.Status},
	})
}

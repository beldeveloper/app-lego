package branch

import (
	"context"
	"fmt"
	"github.com/beldeveloper/app-lego/model"
	"github.com/beldeveloper/go-errors-context"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"log"
	"os"
	"strconv"
	"strings"
)

// NewPostgres creates a new instance of the branches service.
func NewPostgres(conn *pgxpool.Pool, schema model.PgSchema, workDir model.FilePath) Service {
	return Postgres{
		conn:        conn,
		schema:      string(schema),
		branchesDir: string(workDir + "/" + model.BranchesDir),
	}
}

// Postgres implements the branches service with the Postgres storage.
type Postgres struct {
	conn        *pgxpool.Pool
	schema      string
	branchesDir string
}

// Sync takes the branches from VCS and updates the database according to the actual list.
func (p Postgres) Sync(ctx context.Context, r model.Repository, branches []model.Branch) ([]model.Branch, error) {
	old, err := p.FindByRepository(ctx, r)
	if err != nil {
		return nil, errors.WrapContext(err, errors.Context{
			Path:   "service.branch.postgres.Sync: find",
			Params: errors.Params{"repository": r.ID},
		})
	}
	oldMap := make(map[string]model.Branch)
	for _, b := range old {
		oldMap[b.Name] = b
	}
	res := make([]model.Branch, 0, len(branches))
	keepMap := make(map[uint64]bool)
	for _, b := range branches {
		oldBranch, exists := oldMap[b.Name]
		if !exists {
			b.Status = model.BranchStatusPending
			b, err = p.Add(ctx, b)
			if err != nil {
				return nil, errors.WrapContext(err, errors.Context{
					Path:   "service.branch.postgres.Sync: add",
					Params: errors.Params{"branchName": b.Name},
				})
			}
			res = append(res, b)
			continue
		}
		keepMap[oldBranch.ID] = true
		if b.Hash == oldBranch.Hash {
			continue
		}
		b.ID = oldBranch.ID
		b.Status = model.BranchStatusPending
		b, err = p.Update(ctx, b)
		if err != nil {
			return nil, errors.WrapContext(err, errors.Context{
				Path:   "service.branch.postgres.Sync: update",
				Params: errors.Params{"branch": b.ID},
			})
		}
		res = append(res, b)
	}
	del := make([]uint64, 0, len(old))
	for _, b := range old {
		if !keepMap[b.ID] {
			del = append(del, b.ID)
		}
	}
	err = p.DeleteByIDs(ctx, del)
	if err != nil {
		return nil, errors.WrapContext(err, errors.Context{
			Path:   "service.branch.postgres.Sync: delete",
			Params: errors.Params{"ids": del},
		})
	}
	return res, nil
}

// FindAll returns all branches.
func (p Postgres) FindAll(ctx context.Context) ([]model.Branch, error) {
	q := fmt.Sprintf(
		`SELECT "id", "repository_id", "type", "name", "hash", "status" FROM "%s"."branches" ORDER BY "name"`,
		p.schema,
	)
	rows, err := p.conn.Query(ctx, q)
	if err != nil {
		return nil, errors.WrapContext(err, errors.Context{Path: "service.branch.postgres.FindAll: query"})
	}
	defer rows.Close()
	res := make([]model.Branch, 0)
	var b model.Branch
	for rows.Next() {
		err = rows.Scan(&b.ID, &b.RepositoryID, &b.Type, &b.Name, &b.Hash, &b.Status)
		if err != nil {
			return nil, errors.WrapContext(err, errors.Context{Path: "service.branch.postgres.FindAll: scan"})
		}
		res = append(res, b)
	}
	return res, nil
}

// FindByIDs returns all branches with the specific IDs.
func (p Postgres) FindByIDs(ctx context.Context, ids []uint64) ([]model.Branch, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	idsStr := make([]string, len(ids))
	for i, id := range ids {
		idsStr[i] = strconv.Itoa(int(id))
	}
	q := fmt.Sprintf(
		`SELECT "id", "repository_id", "type", "name", "hash", "status" FROM "%s"."branches" WHERE "id" IN (%s)`,
		p.schema,
		strings.Join(idsStr, ","),
	)
	rows, err := p.conn.Query(ctx, q)
	if err != nil {
		return nil, errors.WrapContext(err, errors.Context{
			Path:   "service.branch.postgres.FindByIDs: query",
			Params: errors.Params{"ids": ids},
		})
	}
	defer rows.Close()
	res := make([]model.Branch, 0)
	var b model.Branch
	for rows.Next() {
		err = rows.Scan(&b.ID, &b.RepositoryID, &b.Type, &b.Name, &b.Hash, &b.Status)
		if err != nil {
			return nil, errors.WrapContext(err, errors.Context{
				Path:   "service.branch.postgres.FindByIDs: scan",
				Params: errors.Params{"ids": ids},
			})
		}
		res = append(res, b)
	}
	return res, nil
}

// FindByRepository returns all branches that belong to the specific repository.
func (p Postgres) FindByRepository(ctx context.Context, r model.Repository) ([]model.Branch, error) {
	q := fmt.Sprintf(
		`SELECT "id", "repository_id", "type", "name", "hash", "status" FROM "%s"."branches" WHERE "repository_id" = $1`,
		p.schema,
	)
	rows, err := p.conn.Query(ctx, q, r.ID)
	if err != nil {
		return nil, errors.WrapContext(err, errors.Context{
			Path:   "service.branch.postgres.FindByRepository: query",
			Params: errors.Params{"repository": r.ID},
		})
	}
	defer rows.Close()
	res := make([]model.Branch, 0)
	var b model.Branch
	for rows.Next() {
		err = rows.Scan(&b.ID, &b.RepositoryID, &b.Type, &b.Name, &b.Hash, &b.Status)
		if err != nil {
			return nil, errors.WrapContext(err, errors.Context{
				Path:   "service.branch.postgres.FindByRepository: scan",
				Params: errors.Params{"repository": r.ID},
			})
		}
		res = append(res, b)
	}
	return res, nil
}

// FindByID returns the one branch with the specific ID.
func (p Postgres) FindByID(ctx context.Context, id uint64) (model.Branch, error) {
	var b model.Branch
	q := fmt.Sprintf(
		`SELECT "id", "repository_id", "type", "name", "hash", "status" FROM "%s"."branches" WHERE "id" = $1`,
		p.schema,
	)
	err := p.conn.QueryRow(ctx, q, id).Scan(&b.ID, &b.RepositoryID, &b.Type, &b.Name, &b.Hash, &b.Status)
	if err == pgx.ErrNoRows {
		err = model.ErrNotFound
	}
	return b, errors.WrapContext(err, errors.Context{
		Path:   "service.branch.postgres.FindByID: scan",
		Params: errors.Params{"branch": id},
	})
}

// FindEnqueued returns the one branch that is enqueued.
func (p Postgres) FindEnqueued(ctx context.Context) (model.Branch, error) {
	var b model.Branch
	q := fmt.Sprintf(
		`SELECT "id", "repository_id", "type", "name", "hash", "status" FROM "%s"."branches" WHERE "status" = $1 LIMIT 1`,
		p.schema,
	)
	err := p.conn.QueryRow(ctx, q, model.BranchStatusEnqueued).
		Scan(&b.ID, &b.RepositoryID, &b.Type, &b.Name, &b.Hash, &b.Status)
	if err == pgx.ErrNoRows {
		err = model.ErrNotFound
	}
	return b, errors.WrapContext(err, errors.Context{Path: "service.branch.postgres.FindEnqueued: scan"})
}

// Add saves a new branch.
func (p Postgres) Add(ctx context.Context, b model.Branch) (model.Branch, error) {
	q := fmt.Sprintf(
		`INSERT INTO "%s"."branches" ("repository_id", "type", "name", "hash", "status")
		VALUES ($1, $2, $3, $4, $5) RETURNING "id"`,
		p.schema,
	)
	err := p.conn.QueryRow(ctx, q, b.RepositoryID, b.Type, b.Name, b.Hash, b.Status).Scan(&b.ID)
	if err != nil {
		return b, errors.WrapContext(err, errors.Context{Path: "service.branch.postgres.Add: scan"})
	}
	err = os.Mkdir(fmt.Sprintf("%s/%d", p.branchesDir, b.ID), 0755)
	if err != nil {
		return b, errors.WrapContext(err, errors.Context{Path: "service.branch.postgres.Add: make branch dir"})
	}
	return b, nil
}

// Update modifies a specific branch.
func (p Postgres) Update(ctx context.Context, b model.Branch) (model.Branch, error) {
	q := fmt.Sprintf(`UPDATE "%s"."branches" SET "hash" = $2, "status" = $3 WHERE "id" = $1`, p.schema)
	_, err := p.conn.Exec(ctx, q, b.ID, b.Hash, b.Status)
	return b, errors.WrapContext(err, errors.Context{
		Path:   "service.branch.postgres.Update: exec",
		Params: errors.Params{"branch": b.ID, "status": b.Status, "hash": b.Hash},
	})
}

// DeleteByIDs deletes all branches with the specific ids.
func (p Postgres) DeleteByIDs(ctx context.Context, ids []uint64) error {
	if len(ids) == 0 {
		return nil
	}
	idsStr := make([]string, len(ids))
	for i, id := range ids {
		idsStr[i] = strconv.Itoa(int(id))
	}
	q := fmt.Sprintf(`DELETE FROM "%s"."branches" WHERE "id" IN (%s)`, p.schema, strings.Join(idsStr, ","))
	_, err := p.conn.Exec(ctx, q)
	if err != nil {
		return errors.WrapContext(err, errors.Context{
			Path:   "service.branch.postgres.DeleteByIDs: exec",
			Params: errors.Params{"ids": ids},
		})
	}
	for _, id := range idsStr {
		err = os.RemoveAll(p.branchesDir + "/" + id)
		if err != nil {
			log.Println(errors.WrapContext(err, errors.Context{
				Path:   "service.branch.postgres.DeleteByIDs: remove branch dir",
				Params: errors.Params{"id": id, "dir": p.branchesDir + "/" + id},
			}))
		}
	}
	return nil
}

// UpdateStatus modifies the branch status.
func (p Postgres) UpdateStatus(ctx context.Context, b model.Branch) error {
	q := fmt.Sprintf(`UPDATE "%s"."branches" SET "status" = $2 WHERE "id" = $1`, p.schema)
	_, err := p.conn.Exec(ctx, q, b.ID, b.Status)
	return errors.WrapContext(err, errors.Context{
		Path:   "service.branch.postgres.UpdateStatus: exec",
		Params: errors.Params{"branch": b.ID, "status": b.Status},
	})
}

// LoadComposeData reads the composing configuration for the specific branch.
func (p Postgres) LoadComposeData(ctx context.Context, b model.Branch) ([]byte, error) {
	var data []byte
	q := fmt.Sprintf(`SELECT "compose" FROM "%s"."branches" WHERE "id" = $1`, p.schema)
	err := p.conn.QueryRow(ctx, q, b.ID).Scan(&data)
	if err == pgx.ErrNoRows {
		err = model.ErrNotFound
	}
	return data, errors.WrapContext(err, errors.Context{
		Path:   "service.branch.postgres.LoadComposeData: scan",
		Params: errors.Params{"branch": b.ID},
	})
}

// SaveComposeData saves the composing configuration for the specific branch.
func (p Postgres) SaveComposeData(ctx context.Context, b model.Branch, data []byte) error {
	q := fmt.Sprintf(`UPDATE "%s"."branches" SET "compose" = $2 WHERE "id" = $1`, p.schema)
	_, err := p.conn.Exec(ctx, q, b.ID, data)
	return errors.WrapContext(err, errors.Context{
		Path:   "service.branch.postgres.SaveComposeData: exec",
		Params: errors.Params{"branch": b.ID},
	})
}

package branch

import (
	"context"
	"fmt"
	"github.com/beldeveloper/app-lego/model"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"gopkg.in/yaml.v2"
	"strconv"
	"strings"
)

// NewPostgres creates a new instance of the branches service.
func NewPostgres(conn *pgxpool.Pool, schema string) Postgres {
	return Postgres{conn: conn, schema: schema}
}

// Postgres implements the branches service with the Postgres storage.
type Postgres struct {
	conn   *pgxpool.Pool
	schema string
}

// Sync takes the branches from VCS and updates the database according to the actual list.
func (p Postgres) Sync(ctx context.Context, r model.Repository, branches []model.Branch) ([]model.Branch, error) {
	old, err := p.FindByRepository(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("service.branch.postgres.Sync: find old: %w", err)
	}
	oldMap := make(map[string]*model.Branch)
	for _, b := range old {
		oldMap[b.Name] = &b
	}
	res := make([]model.Branch, 0, len(branches))
	keepMap := make(map[uint64]bool)
	for _, b := range branches {
		oldBranch := oldMap[b.Name]
		if oldBranch == nil {
			b.Status = model.BranchStatusPending
			b, err = p.Add(ctx, b)
			if err != nil {
				return nil, fmt.Errorf("service.branch.postgres.Sync: add: %w", err)
			}
			res = append(res, b)
			continue
		}
		keepMap[oldBranch.ID] = true
		if b.Hash == oldBranch.Hash {
			continue
		}
		b.ID = oldBranch.ID
		b, err = p.Update(ctx, b)
		if err != nil {
			return nil, fmt.Errorf("service.branch.postgres.Sync: update: %w", err)
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
		return nil, fmt.Errorf("service.branch.postgres.Sync: delete: %w", err)
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
		return nil, fmt.Errorf("service.branch.postgres.FindByRepository: query: %w", err)
	}
	defer rows.Close()
	res := make([]model.Branch, 0)
	var b model.Branch
	for rows.Next() {
		err = rows.Scan(&b.ID, &b.RepositoryID, &b.Type, &b.Name, &b.Hash, &b.Status)
		if err != nil {
			return nil, fmt.Errorf("service.branch.postgres.FindByRepository: scan: %w", err)
		}
		res = append(res, b)
	}
	return res, nil
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
	if err != nil {
		if err == pgx.ErrNoRows {
			return b, model.ErrNotFound
		}
		return b, fmt.Errorf("service.branch.postgres.FindEnqueued: query: %w", err)
	}
	return b, nil
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
		return b, fmt.Errorf("service.branch.postgres.Add: insert: %w", err)
	}
	return b, nil
}

// Update modifies a specific branch.
func (p Postgres) Update(ctx context.Context, b model.Branch) (model.Branch, error) {
	q := fmt.Sprintf(`UPDATE "%s"."branches" SET "hash" = $2, "status" = $3 WHERE "id" = $1`, p.schema)
	_, err := p.conn.Exec(ctx, q, b.ID, b.Hash, b.Status)
	if err != nil {
		return b, fmt.Errorf("service.branch.postgres.Update: exec: %w", err)
	}
	return b, nil
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
		return fmt.Errorf("service.branch.postgres.DeleteByIDs: exec: %w", err)
	}
	return nil
}

// UpdateStatus modifies the branch status.
func (p Postgres) UpdateStatus(ctx context.Context, b model.Branch) error {
	q := fmt.Sprintf(`UPDATE "%s"."branches" SET "status" = $2 WHERE "id" = $1`, p.schema)
	_, err := p.conn.Exec(ctx, q, b.ID, b.Status)
	if err != nil {
		return fmt.Errorf("service.branch.postgres.UpdateStatus: exec: %w", err)
	}
	return nil
}

// LoadDockerCompose reads the composing configuration for the specific branch.
func (p Postgres) LoadDockerCompose(ctx context.Context, b model.Branch) (model.DockerCompose, error) {
	var res model.DockerCompose
	var data []byte
	q := fmt.Sprintf(`SELECT "compose" FROM "%s"."branches" WHERE "id" = $1`, p.schema)
	err := p.conn.QueryRow(ctx, q, b.ID).Scan(&data)
	if err != nil {
		if err == pgx.ErrNoRows {
			return res, model.ErrNotFound
		}
		return res, fmt.Errorf("service.branch.postgres.LoadDockerCompose: query: %w", err)
	}
	err = yaml.Unmarshal(data, &res)
	if err != nil {
		return res, fmt.Errorf("service.branch.postgres.LoadDockerCompose: unmarshal: %w", err)
	}
	return res, nil
}

// SaveDockerCompose saves the composing configuration for the specific branch.
func (p Postgres) SaveDockerCompose(ctx context.Context, b model.Branch, dc model.DockerCompose) error {
	data, err := yaml.Marshal(dc)
	if err != nil {
		return fmt.Errorf("service.branch.postgres.SaveDockerCompose: marshal: %w", err)
	}
	q := fmt.Sprintf(`UPDATE "%s"."branches" SET "compose" = $2 WHERE "id" = $1`, p.schema)
	_, err = p.conn.Exec(ctx, q, b.ID, data)
	if err != nil {
		return fmt.Errorf("service.branch.postgres.SaveDockerCompose: exec: %w", err)
	}
	return nil
}

package workqueue

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/keegancsmith/sqlf"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/store/base"
	"github.com/sourcegraph/sourcegraph/internal/db/dbutil"
)

type Store struct {
	*base.Store
	options StoreOptions
}

type StoreOptions struct {
	TableName            string
	ViewName             string
	ColumnExpressions    []*sqlf.Query
	Scan                 RecordScanFn
	OrderByExpression    *sqlf.Query
	AdditionalConditions []*sqlf.Query
	StalledMaxAge        time.Duration
	MaxNumResets         int
}

type RecordScanFn func(rows *sql.Rows, err error) (interface{}, bool, error)

func NewStore(db dbutil.DB, options StoreOptions) *Store {
	if options.ViewName == "" {
		options.ViewName = options.TableName
	}

	return &Store{
		Store:   base.NewWithHandle(db),
		options: options,
	}
}

func (s *Store) Transact(ctx context.Context) (*Store, bool, error) {
	txBase, started, err := s.Store.Transact(ctx)
	if err != nil {
		return nil, false, err
	}

	txStore := &Store{
		Store:   txBase,
		options: s.options,
	}

	return txStore, started, err
}

func (s *Store) Dequeue(ctx context.Context) (interface{}, *Store, bool, error) {
	query := sqlf.Sprintf(
		selectCandidateQuery,
		quote(s.options.ViewName),
		makeConditionSuffix(s.options.AdditionalConditions),
		s.options.OrderByExpression,
		quote(s.options.TableName),
	)

	for {
		id, ok, err := base.ScanFirstInt(s.Query(ctx, query))
		if err != nil {
			return nil, nil, false, err
		}
		if !ok {
			return nil, nil, false, nil
		}

		record, tx, ok, err := s.lock(ctx, id)
		if err != nil {
			if err == ErrDequeueRace {
				continue
			}

			return nil, nil, false, err
		}

		return record, tx, ok, nil
	}
}

const selectCandidateQuery = `
-- source: enterprise/internal/codeintel/store/workqueue/store.go:Dequeue
WITH candidate AS (
	SELECT id FROM %s
	WHERE
		state = 'queued' AND
		(process_after IS NULL OR process_after >= NOW())
		%s
	ORDER BY %s
	FOR UPDATE SKIP LOCKED
	LIMIT 1
)
UPDATE %s
SET
	state = 'processing',
	started_at = now()
WHERE id IN (SELECT id FROM candidate)
RETURNING id
`

func (s *Store) lock(ctx context.Context, id int) (interface{}, *Store, bool, error) {
	tx, started, err := s.Transact(ctx)
	if err != nil {
		return nil, nil, false, err
	}
	if !started {
		return nil, nil, false, ErrDequeueTransaction
	}

	_, exists, err := base.ScanFirstInt(tx.Query(ctx, sqlf.Sprintf(
		lockQuery,
		quote(s.options.TableName),
		id,
	)))
	if err != nil {
		return nil, nil, false, tx.Done(err)
	}
	if !exists {
		return nil, nil, false, tx.Done(ErrDequeueRace)
	}

	record, exists, err := s.options.Scan(tx.Query(ctx, sqlf.Sprintf(
		selectRecordQuery,
		sqlf.Join(s.options.ColumnExpressions, ", "),
		quote(s.options.ViewName),
		id,
	)))
	if err != nil {
		fmt.Printf(":( %s\n", sqlf.Sprintf(
			selectRecordQuery,
			sqlf.Join(s.options.ColumnExpressions, ", "),
			quote(s.options.ViewName),
			id,
		).Query(sqlf.PostgresBindVar))
		return nil, nil, false, tx.Done(err)
	}
	if !exists {
		return nil, nil, false, tx.Done(ErrNoRecord)
	}

	return record, tx, true, nil
}

const lockQuery = `
-- source: enterprise/internal/codeintel/store/workqueue/store.go:lock
SELECT 1 FROM %s
WHERE id = %s
FOR UPDATE SKIP LOCKED
LIMIT 1
`

// TODO - remove the alias here (or standardize it)
const selectRecordQuery = `
-- source: enterprise/internal/codeintel/store/workqueue/store.go:lock
SELECT %s FROM %s
WHERE id = %s
LIMIT 1
`

func (s *Store) Requeue(ctx context.Context, id int, after time.Time) error {
	return s.QueryForEffect(ctx, sqlf.Sprintf(
		requeueQuery,
		quote(s.options.TableName),
		after,
		id,
	))
}

const requeueQuery = `
-- source: enterprise/internal/codeintel/store/workqueue/store.go:Requeue
UPDATE %s
SET state = 'queued', process_after = %s
WHERE id = %s
`

func (s *Store) ResetStalled(ctx context.Context, now time.Time) (resetIDs, erroredIDs []int, err error) {
	resetIDs, err = s.resetStalled(ctx, resetStalledQuery, now)
	if err != nil {
		return nil, nil, err
	}

	erroredIDs, err = s.resetStalled(ctx, resetStalledMaxResetsQuery, now)
	if err != nil {
		return nil, nil, err
	}

	return resetIDs, erroredIDs, nil
}

func (s *Store) resetStalled(ctx context.Context, q string, now time.Time) ([]int, error) {
	return base.ScanInts(s.Query(
		ctx,
		sqlf.Sprintf(
			q,
			quote(s.options.TableName),
			now.UTC(),
			s.options.StalledMaxAge/time.Second,
			s.options.MaxNumResets,
			quote(s.options.TableName),
		),
	))
}

const resetStalledQuery = `
-- source: enterprise/internal/codeintel/store/workqueue/store.go:ResetStalled
WITH stalled AS (
	SELECT id FROM %s
	WHERE
		state = 'processing' AND
		%s - started_at > (%s * interval '1 second') AND
		num_resets < %s
	FOR UPDATE SKIP LOCKED
)
UPDATE %s
SET
	state = 'queued',
	started_at = null,
	num_resets = num_resets + 1
WHERE id IN (SELECT id FROM stalled)
RETURNING id
`

const resetStalledMaxResetsQuery = `
-- source: enterprise/internal/codeintel/store/workqueue/store.go:ResetStalled
WITH stalled AS (
	SELECT id FROM %s
	WHERE
		state = 'processing' AND
		%s - started_at > (%s * interval '1 second') AND
		num_resets >= %s
	FOR UPDATE SKIP LOCKED
)
UPDATE %s
SET
	state = 'errored',
	finished_at = clock_timestamp(),
	failure_message = 'failed to process'
WHERE id IN (SELECT id FROM stalled)
RETURNING id
`

func quote(s string) *sqlf.Query {
	return sqlf.Sprintf(s)
}

func makeConditionSuffix(conditions []*sqlf.Query) *sqlf.Query {
	if len(conditions) == 0 {
		return sqlf.Sprintf("")
	}

	var quotedConditions []*sqlf.Query
	for _, condition := range conditions {
		// Ensure everything is quoted in case a condition has an OR
		quotedConditions = append(quotedConditions, sqlf.Sprintf("(%s)", condition))
	}

	return sqlf.Sprintf("AND %s", sqlf.Join(quotedConditions, " AND "))
}

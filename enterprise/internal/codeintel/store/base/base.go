package base

import (
	"context"
	"database/sql"

	"github.com/hashicorp/go-multierror"
	"github.com/keegancsmith/sqlf"
	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/internal/db/dbutil"
)

type Store struct {
	db dbutil.DB
}

// TODO
type ShareableStore interface {
	Handle() dbutil.DB
}

var _ ShareableStore = &Store{}

func New(postgresDSN string, app string) (*Store, error) {
	db, err := dbutil.NewDB(postgresDSN, app)
	if err != nil {
		return nil, err
	}

	return &Store{db: db}, nil
}

func NewWithHandle(db dbutil.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Handle() dbutil.DB {
	return s.db
}

func (s *Store) Use(other ShareableStore) *Store {
	return &Store{db: other.Handle()}
}

func (s *Store) Query(ctx context.Context, query *sqlf.Query) (*sql.Rows, error) {
	return s.db.QueryContext(ctx, query.Query(sqlf.PostgresBindVar), query.Args()...)
}

func (s *Store) QueryForEffect(ctx context.Context, query *sqlf.Query) error {
	rows, err := s.Query(ctx, query)
	if err != nil {
		return err
	}
	return CloseRows(rows, nil)
}

func (s *Store) InTransaction() bool {
	_, ok := s.db.(dbutil.Tx)
	return ok
}

func (s *Store) Transact(ctx context.Context) (*Store, bool, error) {
	if s.InTransaction() {
		// Already in a Tx
		return s, false, nil
	}

	tb, ok := s.db.(dbutil.TxBeginner)
	if !ok {
		// Not a Tx nor a TxBeginner
		return nil, false, ErrNotTransactable
	}

	tx, err := tb.BeginTx(ctx, nil)
	if err != nil {
		return nil, false, errors.Wrap(err, "store: BeginTx")
	}

	return &Store{db: tx}, true, nil
}

func (s *Store) Done(err error) error {
	if tx, ok := s.db.(dbutil.Tx); ok {
		if err != nil {
			if rollErr := tx.Rollback(); rollErr != nil {
				err = multierror.Append(err, rollErr)
			}
		} else {
			if commitErr := tx.Commit(); commitErr != nil {
				err = multierror.Append(err, commitErr)
			}
		}
	}

	return err
}

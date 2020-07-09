package base

import (
	"context"
	"database/sql"

	"github.com/hashicorp/go-multierror"
	"github.com/keegancsmith/sqlf"
	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/internal/db/dbutil"
)

// Store is an abstract Postgres-backed data access layer. Instances of this struct
// should not be used directly, but should be used compositionally by other stores
// that implement logic specific to a domain.
//
// The following is a minimal example of decorating the base store that preserves
// the correct behavior of the underlying base. Note that `Use` and `Transact` must
// be re-defined in the outer layer in order to create a useful return value. Failure
// to re-define these methods will result in `Use` and `Transact` methods that return
// a modified base store with no methods from the outer layer. All other methods of
// the base store are available on the outer layer without needing to be re-defined.
//
//     type SprocketStore struct {
//         *base.Store
//     }
//
//     func NewWithHandle(db dbutil.DB) *SprocketStore {
//         return &SprocketStore{Store: base.NewWithHandle(db)}
//     }
//
//     func (s *SprocketStore) Use(other base.ShareableStore) *SprocketStore {
//         return &SprocketStore{Store: s.Store.Use(other)}
//     }
//
//     func (s *SprocketStore) Transact(ctx context.Context) (*SprocketStore, bool, error) {
//         txBase, started, err := s.Store.Transact(ctx)
//         if err != nil {
//              return nil, false, err
//         }
//
//         return &SprocketStore{Store: txBase}, started, nil
//     }
type Store struct {
	db dbutil.DB
}

// ShareableStore is implemented by stores to explicitly allow distinct store instances
// to reference the store's underlying handle. This is used to share transactions between
// multiple stores. See `Store.Use` for additional details.
type ShareableStore interface {
	// Handle returns the underlying database handle.
	Handle() dbutil.DB
}

var _ ShareableStore = &Store{}

// New returns a new *Store connected to the given dsn (data store name).
func New(postgresDSN, app string) (*Store, error) {
	db, err := dbutil.NewDB(postgresDSN, app)
	if err != nil {
		return nil, err
	}

	return &Store{db: db}, nil
}

// NewWithHandle returns a new *Store using the given database handle.
func NewWithHandle(db dbutil.DB) *Store {
	return &Store{db: db}
}

// Handle returns the underlying database handle.
func (s *Store) Handle() dbutil.DB {
	return s.db
}

// Use creates a new store with the underlying database handle from the given store. This
// method should be used when two distinct store instances need to perform an operation
// within the same shared transaction.
//
//     txn1 := store1.Transact(ctx) // Creates a transaction
//     txn2 := store2.Use(txn1)     // References the same transaction
//     txn1.A(ctx) // Occurs within shared transaction
//     txn2.B(ctx) // Occurs within shared transaction
//     txn1.Done() // closes shared transaction
func (s *Store) Use(other ShareableStore) *Store {
	return &Store{db: other.Handle()}
}

// Query performs QueryContext on the underlying connection.
func (s *Store) Query(ctx context.Context, query *sqlf.Query) (*sql.Rows, error) {
	return s.db.QueryContext(ctx, query.Query(sqlf.PostgresBindVar), query.Args()...)
}

// QueryForEffect performs a query and throws away the result.
func (s *Store) QueryForEffect(ctx context.Context, query *sqlf.Query) error {
	rows, err := s.Query(ctx, query)
	if err != nil {
		return err
	}
	return CloseRows(rows, nil)
}

// InTransaction returns true if the underlying database handel is in a transaction.
func (s *Store) InTransaction() bool {
	_, ok := s.db.(dbutil.Tx)
	return ok
}

// Transact returns a modified store whose methods operate within the context of a
// transaction. This method will return an error if the underlying store cannot be
// interface upgraded to a TxBeginner. If this method results in a new transaction
// starting, a true-valued flag is returned.
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

// Done commits underlying the transaction on a nil error value and performs a rollback
// otherwise. The resulting error value is a multierror containing the error parameter
// along with any error that occurs during rollback or commit of the transaction. If the
// store does not wrap a transaction the original error value is returned unchanged.
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

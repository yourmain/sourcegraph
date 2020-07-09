package store

import "context"

// DoneFunc is the function type of store's Done method.
type DoneFunc func(err error) error

// noopDoneFunc is a behaviorless DoneFunc.
func noopDoneFunc(err error) error {
	return err
}

// Transact returns a store whose methods operate within the context of a transaction.
// This method will return an error if the underlying store cannot be interface upgraded
// to a TxBeginner.
func (s *store) Transact(ctx context.Context) (Store, error) {
	return s.transact(ctx)
}

// Transact returns a store whose methods operate within the context of a transaction.
// This method will return an error if the underlying store cannot be interface upgraded
// to a TxBeginner.
func (s *store) transact(ctx context.Context) (*store, error) {
	tx, err := s.Store.Transact(ctx)
	return &store{Store: tx}, err
}

// // transact returns a store whose methods operate within the context of a transaction.
// // This method also returns a boolean flag indicating whether a new transaction was created.
// func (s *store) transact(ctx context.Context) (*store, bool, error) {
// 	started := !s.InTransaction()

// 	txBase, err := s.Store.Transact(ctx)
// 	if err != nil {
// 		return nil, false, err
// 	}

// 	return &store{Store: txBase}, started, nil
// }

// // Savepoint creates a named position in the transaction from which all additional work
// // can be discarded. The returned identifier can be passed to RollbackToSavepont to undo
// // all the work since this call.
// func (s *store) Savepoint(ctx context.Context) (string, error) {
// 	if !s.Store.InTransaction() {
// 		return "", ErrNoTransaction
// 	}

// 	id, err := uuid.NewRandom()
// 	if err != nil {
// 		return "", err
// 	}

// 	savepointID := fmt.Sprintf("sp_%s", strings.ReplaceAll(id.String(), "-", "_"))
// 	s.savepointIDs = append(s.savepointIDs, savepointID)

// 	// Unfortunately, it's a syntax error to supply this as a param
// 	if err := s.queryForEffect(ctx, sqlf.Sprintf("SAVEPOINT "+savepointID)); err != nil {
// 		return "", err
// 	}

// 	return savepointID, nil
// }

// // RollbackToSavepoint throws away all the work on the underlying transaction since the
// // savepoint with the given name was created.
// func (s *store) RollbackToSavepoint(ctx context.Context, savepointID string) error {
// 	if !s.Store.InTransaction() {
// 		return ErrNoTransaction
// 	}

// 	for i, id := range s.savepointIDs {
// 		if savepointID != id {
// 			continue
// 		}

// 		// Pop this and all later savepoints
// 		s.savepointIDs = s.savepointIDs[:i]

// 		// Unfortunately, it's a syntax error to supply this as a param
// 		return s.queryForEffect(ctx, sqlf.Sprintf("ROLLBACK TO SAVEPOINT "+savepointID))
// 	}

// 	return ErrNoSavepoint
// }

package store

import "context"

// DoneFunc is the function type of store's Done method.
type DoneFunc func(err error) error

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

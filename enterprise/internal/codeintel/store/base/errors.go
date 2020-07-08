package base

import "errors"

// ErrNotTransactable occurs when Transact is called on a store whose underlying
// store handle does not support beginning a transaction.
var ErrNotTransactable = errors.New("store: not transactable")

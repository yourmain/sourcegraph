package store

// func TestSavepointNotInTransaction(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip()
// 	}
// 	dbtesting.SetupGlobalTestDB(t)
// 	store := testStore()

// 	if _, err := store.Savepoint(context.Background()); err != ErrNoTransaction {
// 		t.Errorf("unexpected error. want=%q have=%q", ErrNoTransaction, err)
// 	}
// }

// func TestRollbackToSavepointNotInTransaction(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip()
// 	}
// 	dbtesting.SetupGlobalTestDB(t)
// 	store := testStore()

// 	if err := store.RollbackToSavepoint(context.Background(), "sp_test"); err != ErrNoTransaction {
// 		t.Errorf("unexpected error. want=%q have=%q", ErrNoTransaction, err)
// 	}
// }

// func TestRollbackToSavepointTwice(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip()
// 	}
// 	dbtesting.SetupGlobalTestDB(t)
// 	store := testStore()

// 	tx, err := store.Transact(context.Background())
// 	if err != nil {
// 		t.Errorf("unexpected error creating transaction: %s", err)
// 	}
// 	defer func() { _ = tx.Done(nil) }()

// 	savepointID, err := tx.Savepoint(context.Background())
// 	if err != nil {
// 		t.Errorf("unexpected error creating savepoint: %s", err)
// 	}

// 	if err := tx.RollbackToSavepoint(context.Background(), savepointID); err != nil {
// 		t.Errorf("unexpected error rolling back to savepoint: %s", err)
// 	}

// 	if err := tx.RollbackToSavepoint(context.Background(), savepointID); err != ErrNoSavepoint {
// 		t.Errorf("unexpected error. want=%q have=%q", ErrNoSavepoint, err)
// 	}
// }

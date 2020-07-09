package workqueue

import (
	"context"
	"database/sql"
	"sort"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/keegancsmith/sqlf"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/store/base"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
)

func init() {
	dbtesting.DBNameSuffix = "workqueue"
}

func TestDequeueState(t *testing.T) {
	setupStoreTest(t)

	if _, err := dbconn.Global.Exec(`
		INSERT INTO workqueue_test (id, state, uploaded_at)
		VALUES
			(1, 'queued', NOW() - '1 minute'::interval),
			(2, 'queued', NOW() - '2 minute'::interval),
			(3, 'state2', NOW() - '3 minute'::interval),
			(4, 'queued', NOW() - '4 minute'::interval),
			(5, 'state2', NOW() - '5 minute'::interval)
	`); err != nil {
		t.Fatalf("unexpected error inserting records: %s", err)
	}

	record, tx, ok, err := testStore(defaultTestStoreOptions).Dequeue(context.Background(), nil)
	assertDequeueRecordResult(t, 4, record, tx, ok, err)
}

func TestDequeueOrder(t *testing.T) {
	setupStoreTest(t)

	if _, err := dbconn.Global.Exec(`
		INSERT INTO workqueue_test (id, state, uploaded_at)
		VALUES
			(1, 'queued', NOW() - '2 minute'::interval),
			(2, 'queued', NOW() - '5 minute'::interval),
			(3, 'queued', NOW() - '3 minute'::interval),
			(4, 'queued', NOW() - '1 minute'::interval),
			(5, 'queued', NOW() - '4 minute'::interval)
	`); err != nil {
		t.Fatalf("unexpected error inserting records: %s", err)
	}

	record, tx, ok, err := testStore(defaultTestStoreOptions).Dequeue(context.Background(), nil)
	assertDequeueRecordResult(t, 2, record, tx, ok, err)
}

func TestDequeueConditions(t *testing.T) {
	setupStoreTest(t)

	if _, err := dbconn.Global.Exec(`
		INSERT INTO workqueue_test (id, state, uploaded_at)
		VALUES
			(1, 'queued', NOW() - '1 minute'::interval),
			(2, 'queued', NOW() - '2 minute'::interval),
			(3, 'queued', NOW() - '3 minute'::interval),
			(4, 'queued', NOW() - '4 minute'::interval),
			(5, 'queued', NOW() - '5 minute'::interval)
	`); err != nil {
		t.Fatalf("unexpected error inserting records: %s", err)
	}

	conditions := []*sqlf.Query{sqlf.Sprintf("w.id < 4")}
	record, tx, ok, err := testStore(defaultTestStoreOptions).Dequeue(context.Background(), conditions)
	assertDequeueRecordResult(t, 3, record, tx, ok, err)
}

func TestDequeueDelay(t *testing.T) {
	setupStoreTest(t)

	if _, err := dbconn.Global.Exec(`
		INSERT INTO workqueue_test (id, state, uploaded_at, process_after)
		VALUES
			(1, 'queued', NOW() - '1 minute'::interval, NULL),
			(2, 'queued', NOW() - '2 minute'::interval, NULL),
			(3, 'queued', NOW() - '3 minute'::interval, NOW() + '2 minute'::interval),
			(4, 'queued', NOW() - '4 minute'::interval, NULL),
			(5, 'queued', NOW() - '5 minute'::interval, NOW() + '1 minute'::interval)
	`); err != nil {
		t.Fatalf("unexpected error inserting records: %s", err)
	}

	record, tx, ok, err := testStore(defaultTestStoreOptions).Dequeue(context.Background(), nil)
	assertDequeueRecordResult(t, 4, record, tx, ok, err)
}

func TestDequeueView(t *testing.T) {
	setupStoreTest(t)

	if _, err := dbconn.Global.Exec(`
		INSERT INTO workqueue_test (id, state, uploaded_at)
		VALUES
			(1, 'queued', NOW() - '1 minute'::interval),
			(2, 'queued', NOW() - '2 minute'::interval),
			(3, 'queued', NOW() - '3 minute'::interval),
			(4, 'queued', NOW() - '4 minute'::interval),
			(5, 'queued', NOW() - '5 minute'::interval)
	`); err != nil {
		t.Fatalf("unexpected error inserting records: %s", err)
	}

	options := StoreOptions{
		TableName:         "workqueue_test w",
		ViewName:          "workqueue_test_view v",
		Scan:              testScanFirstRecordView,
		OrderByExpression: sqlf.Sprintf("v.uploaded_at"),
		ColumnExpressions: []*sqlf.Query{
			sqlf.Sprintf("v.id"),
			sqlf.Sprintf("v.state"),
			sqlf.Sprintf("v.new_field"),
		},
		StalledMaxAge: time.Second * 5,
		MaxNumResets:  5,
	}

	conditions := []*sqlf.Query{sqlf.Sprintf("v.new_field < 15")}
	record, tx, ok, err := testStore(options).Dequeue(context.Background(), conditions)
	assertDequeueRecordViewResult(t, 2, 14, record, tx, ok, err)
}

func TestDequeueConcurrent(t *testing.T) {
	setupStoreTest(t)

	if _, err := dbconn.Global.Exec(`
		INSERT INTO workqueue_test (id, state, uploaded_at)
		VALUES
			(1, 'queued', NOW() - '2 minute'::interval),
			(2, 'queued', NOW() - '1 minute'::interval)
	`); err != nil {
		t.Fatalf("unexpected error inserting records: %s", err)
	}

	store := testStore(defaultTestStoreOptions)

	// Worker A
	record1, tx1, ok, err := store.Dequeue(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !ok {
		t.Fatalf("expected a dequeueable record")
	}
	defer func() { _ = tx1.Done(nil) }()

	// Worker B
	record2, tx2, ok, err := store.Dequeue(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !ok {
		t.Fatalf("expected a second dequeueable record")
	}
	defer func() { _ = tx2.Done(nil) }()

	if val := record1.(TestRecord).ID; val != 1 {
		t.Errorf("unexpected id. want=%d have=%d", 1, val)
	}
	if val := record2.(TestRecord).ID; val != 2 {
		t.Errorf("unexpected id. want=%d have=%d", 2, val)
	}

	// Worker C
	_, _, ok, err = store.Dequeue(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if ok {
		t.Fatalf("did not expect a third dequeueable record")
	}
}

func TestRequeue(t *testing.T) {
	setupStoreTest(t)

	if _, err := dbconn.Global.Exec(`
		INSERT INTO workqueue_test (id, state)
		VALUES
			(1, 'processing')
	`); err != nil {
		t.Fatalf("unexpected error inserting records: %s", err)
	}

	after := testNow().Add(time.Hour)

	if err := testStore(defaultTestStoreOptions).Requeue(context.Background(), 1, after); err != nil {
		t.Fatalf("unexpected error requeueing index: %s", err)
	}

	rows, err := dbconn.Global.Query(`SELECT state, process_after FROM workqueue_test WHERE id = 1`)
	if err != nil {
		t.Fatalf("unexpected error querying record: %s", err)
	}
	defer func() { _ = base.CloseRows(rows, nil) }()

	if !rows.Next() {
		t.Fatal("expected record to exist")
	}

	var state string
	var processAfter *time.Time

	if err := rows.Scan(&state, &processAfter); err != nil {
		t.Fatalf("unexpected error scanning record: %s", err)
	}
	if state != "queued" {
		t.Errorf("unexpected state. want=%q have=%q", "queued", state)
	}
	if processAfter == nil || *processAfter != after {
		t.Errorf("unexpected process after. want=%s have=%s", after, processAfter)
	}
}

func TestResetStalled(t *testing.T) {
	setupStoreTest(t)

	if _, err := dbconn.Global.Exec(`
		INSERT INTO workqueue_test (id, state, started_at, num_resets)
		VALUES
			(1, 'processing', NOW() - '6 second'::interval, 1),
			(2, 'processing', NOW() - '2 second'::interval, 0),
			(3, 'processing', NOW() - '3 second'::interval, 0),
			(4, 'processing', NOW() - '8 second'::interval, 0),
			(5, 'processing', NOW() - '8 second'::interval, 0),
			(6, 'processing', NOW() - '6 second'::interval, 5),
			(7, 'processing', NOW() - '8 second'::interval, 5)
	`); err != nil {
		t.Fatalf("unexpected error inserting records: %s", err)
	}

	tx, err := dbconn.Global.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()

	// Row lock upload 5 in a transaction which should be skipped by ResetStalled
	if _, err := tx.Exec(`SELECT * FROM workqueue_test WHERE id = 5 FOR UPDATE`); err != nil {
		t.Fatal(err)
	}

	resetIDs, erroredIDs, err := testStore(defaultTestStoreOptions).ResetStalled(context.Background())
	if err != nil {
		t.Fatalf("unexpected error resetting stalled uploads: %s", err)
	}
	sort.Ints(resetIDs)
	sort.Ints(erroredIDs)

	if diff := cmp.Diff([]int{1, 4}, resetIDs); diff != "" {
		t.Errorf("unexpected reset ids (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff([]int{6, 7}, erroredIDs); diff != "" {
		t.Errorf("unexpected errored ids (-want +got):\n%s", diff)
	}

	rows, err := dbconn.Global.Query(`SELECT state, num_resets FROM workqueue_test WHERE id = 1`)
	if err != nil {
		t.Fatalf("unexpected error querying record: %s", err)
	}
	defer func() { _ = base.CloseRows(rows, nil) }()

	if !rows.Next() {
		t.Fatal("expected record to exist")
	}

	var state string
	var numResets int
	if err := rows.Scan(&state, &numResets); err != nil {
		t.Fatalf("unexpected error scanning record: %s", err)
	}
	if state != "queued" {
		t.Errorf("unexpected state. want=%q have=%q", "queued", state)
	}
	if numResets != 2 {
		t.Errorf("unexpected num resets. want=%d have=%d", 2, numResets)
	}

	rows, err = dbconn.Global.Query(`SELECT state FROM workqueue_test WHERE id = 6`)
	if err != nil {
		t.Fatalf("unexpected error querying record: %s", err)
	}
	defer func() { _ = base.CloseRows(rows, nil) }()

	if !rows.Next() {
		t.Fatal("expected record to exist")
	}

	if err := rows.Scan(&state); err != nil {
		t.Fatalf("unexpected error scanning record: %s", err)
	}
	if state != "errored" {
		t.Errorf("unexpected state. want=%q have=%q", "errored", state)
	}
}

func testStore(options StoreOptions) *Store {
	return NewStore(base.NewHandleWithDB(dbconn.Global), options)
}

type TestRecord struct {
	ID    int
	State string
}

func testScanFirstRecord(rows *sql.Rows, queryErr error) (v interface{}, _ bool, err error) {
	if queryErr != nil {
		return nil, false, queryErr
	}
	defer func() { err = base.CloseRows(rows, err) }()

	if rows.Next() {
		var record TestRecord
		if err := rows.Scan(&record.ID, &record.State); err != nil {
			return nil, false, err
		}

		return record, true, nil
	}

	return nil, false, nil
}

type TestRecordView struct {
	ID       int
	State    string
	NewField int
}

func testScanFirstRecordView(rows *sql.Rows, queryErr error) (v interface{}, exists bool, err error) {
	if queryErr != nil {
		return nil, false, queryErr
	}
	defer func() { err = base.CloseRows(rows, err) }()

	if rows.Next() {
		var record TestRecordView
		if err := rows.Scan(&record.ID, &record.State, &record.NewField); err != nil {
			return nil, false, err
		}

		return record, true, nil
	}

	return nil, false, nil
}

func setupStoreTest(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)

	if _, err := dbconn.Global.Exec(`
		CREATE TABLE IF NOT EXISTS workqueue_test (
			id              integer NOT NULL,
			state           text NOT NULL,
			failure_message text,
			started_at      timestamp with time zone,
			finished_at     timestamp with time zone,
			process_after   timestamp with time zone,
			num_resets      integer NOT NULL default 0,
			uploaded_at     timestamp with time zone NOT NULL default NOW()
		)
	`); err != nil {
		t.Fatalf("unexpected error creating test table: %s", err)
	}

	if _, err := dbconn.Global.Exec(`
		CREATE OR REPLACE VIEW workqueue_test_view AS (
			SELECT w.*, (w.id * 7) as new_field FROM workqueue_test w
		)
	`); err != nil {
		t.Fatalf("unexpected error creating test table: %s", err)
	}
}

var defaultTestStoreOptions = StoreOptions{
	TableName:         "workqueue_test w",
	Scan:              testScanFirstRecord,
	OrderByExpression: sqlf.Sprintf("w.uploaded_at"),
	ColumnExpressions: []*sqlf.Query{
		sqlf.Sprintf("w.id"),
		sqlf.Sprintf("w.state"),
	},
	StalledMaxAge: time.Second * 5,
	MaxNumResets:  5,
}

func assertDequeueRecordResult(t *testing.T, expectedID int, record interface{}, tx *Store, ok bool, err error) {
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !ok {
		t.Fatalf("expected a dequeueable record")
	}
	defer func() { _ = tx.Done(nil) }()

	if val := record.(TestRecord).ID; val != expectedID {
		t.Errorf("unexpected id. want=%d have=%d", expectedID, val)
	}
	if val := record.(TestRecord).State; val != "processing" {
		t.Errorf("unexpected state. want=%s have=%s", "processing", val)
	}
}

func assertDequeueRecordViewResult(t *testing.T, expectedID, expectedNewField int, record interface{}, tx *Store, ok bool, err error) {
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !ok {
		t.Fatalf("expected a dequeueable record")
	}
	defer func() { _ = tx.Done(nil) }()

	if val := record.(TestRecordView).ID; val != expectedID {
		t.Errorf("unexpected id. want=%d have=%d", expectedID, val)
	}
	if val := record.(TestRecordView).State; val != "processing" {
		t.Errorf("unexpected state. want=%s have=%s", "processing", val)
	}
	if val := record.(TestRecordView).NewField; val != expectedNewField {
		t.Errorf("unexpected new field. want=%d have=%d", expectedNewField, val)
	}
}

func testNow() time.Time {
	return time.Now().UTC().Truncate(time.Second)
}

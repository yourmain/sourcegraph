package base

import (
	"database/sql"

	"github.com/hashicorp/go-multierror"
)

// CloseRows closes the given rows object. The resulting error is a multierror
// containing the error parameter along with any errors that occur during scanning
// or closing the rows object. The rows object is assumed to be non-nil.
//
// The signature of this function allows scan methods to be written uniformly:
//
//     func ScanThings(rows *sql.Rows, queryErr error) (_ []Thing, err error) {
//         if queryErr != nil {
//             return nil, queryErr
//         }
//         defer func() { err = CloseRows(rows, err) }()
//
//         // read things from rows
//     }
//
// Scan methods should be called directly with the results of `*store.Query` to
// ensure that the rows are always properly handled.
//
//     thing, err := ScanThings(store.Query(ctx, query))
//
//
func CloseRows(rows *sql.Rows, err error) error {
	if closeErr := rows.Close(); closeErr != nil {
		err = multierror.Append(err, closeErr)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		err = multierror.Append(err, rowsErr)
	}

	return err
}

// ScanStrings reads string values from the given row object.
func ScanStrings(rows *sql.Rows, queryErr error) (_ []string, err error) {
	if queryErr != nil {
		return nil, queryErr
	}
	defer func() { err = CloseRows(rows, err) }()

	var values []string
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}

		values = append(values, value)
	}

	return values, nil
}

// ScanFirstString reads string values from the given row object and returns the first one.
// The query for these rows should apply a LIMIT as all of the rows will be consumed. If no
// rows match the query, a false-valued flag is returned.
func ScanFirstString(rows *sql.Rows, err error) (string, bool, error) {
	values, err := ScanStrings(rows, err)
	if err != nil || len(values) == 0 {
		return "", false, err
	}
	return values[0], true, nil
}

// ScanInts reads integer values from the given row object.
func ScanInts(rows *sql.Rows, queryErr error) (_ []int, err error) {
	if queryErr != nil {
		return nil, queryErr
	}
	defer func() { err = CloseRows(rows, err) }()

	var values []int
	for rows.Next() {
		var value int
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}

		values = append(values, value)
	}

	return values, nil
}

// ScanFirstInt reads integer values from the given row object and returns the first one.
// The query for these rows should apply a LIMIT as all of the rows will be consumed. If no
// rows match the query, a false-valued flag is returned.
func ScanFirstInt(rows *sql.Rows, err error) (int, bool, error) {
	values, err := ScanInts(rows, err)
	if err != nil || len(values) == 0 {
		return 0, false, err
	}
	return values[0], true, nil
}

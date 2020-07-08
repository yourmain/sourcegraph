package base

import (
	"database/sql"

	"github.com/hashicorp/go-multierror"
)

// CloseRows closes the rows object and checks its error value.
func CloseRows(rows *sql.Rows, err error) error {
	if closeErr := rows.Close(); closeErr != nil {
		err = multierror.Append(err, closeErr)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		err = multierror.Append(err, rowsErr)
	}

	return err
}

// ScanStrings scans a slice of strings from the return value of `*Store.Query`.
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

// ScanFirstString scans a slice of strings from the return value of `*Store.Query` and returns the first.
func ScanFirstString(rows *sql.Rows, err error) (string, bool, error) {
	values, err := ScanStrings(rows, err)
	if err != nil || len(values) == 0 {
		return "", false, err
	}
	return values[0], true, nil
}

// ScanInts scans a slice of ints from the return value of `*Store.Query`.
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

// ScanFirstInt scans a slice of ints from the return value of `*Store.Query` and returns the first.
func ScanFirstInt(rows *sql.Rows, err error) (int, bool, error) {
	values, err := ScanInts(rows, err)
	if err != nil || len(values) == 0 {
		return 0, false, err
	}
	return values[0], true, nil
}

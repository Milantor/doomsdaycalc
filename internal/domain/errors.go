package domain

import "errors"

// ErrNotFound: returned by repositories when the record does not exist. Compare it
// with errors.Is.
var ErrNotFound = errors.New("not found")

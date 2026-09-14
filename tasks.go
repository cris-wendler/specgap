// Package specgap carries the task files, so that a build installed
// with go install can pose a task without the repository being checked
// out beside it.
//
// A task cut from another repository still needs that repository. That
// is inherent: the workspace is a copy of it.
package specgap

import "embed"

// Tasks holds the files under tasks/ as they were when this was built.
// The pattern has no all: prefix, which needs a newer Go than this
// project supports, and no task file begins with a dot or an underscore,
// which is what that prefix would change.
//
//go:embed tasks
var Tasks embed.FS

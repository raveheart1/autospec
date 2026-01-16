// Package init provides initialization settings management for autospec.
//
// This package handles reading and writing the .autospec/init.yml file,
// which tracks how autospec was initialized in a project (global vs project
// scope, agent configurations, etc.).
//
// The init.yml file is created by `autospec init` and read by `autospec doctor`
// to determine the correct location for permission checks.
package init

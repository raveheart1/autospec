// Package dag provides DAG (Directed Acyclic Graph) schema parsing, validation,
// and visualization for multi-spec workflow orchestration.
//
// The package supports:
//   - Parsing DAG YAML configuration files with schema_version, dag metadata, and layers
//   - Validating DAG structure including cycle detection and spec reference validation
//   - ASCII visualization of DAG dependencies
//
// DAG files are stored in .autospec/dags/*.yaml and define dependencies between
// multiple spec features that can be processed in parallel across layers.
package dag

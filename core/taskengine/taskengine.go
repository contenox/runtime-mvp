// Package taskengine provides an engine for orchestrating chains of LLM-based tasks.
//
// taskengine enables building AI workflows where tasks are linked in sequence, supporting
// conditional branching, numeric or scored evaluation, range resolution, and optional
// integration with external systems (hooks). Each task can invoke an LLM prompt or a custom
// hook function depending on its type.
//
// Hooks are pluggable interfaces that allow tasks to perform side effects — calling APIs,
// saving data, or triggering custom business logic — outside of prompt-based processing.
//
// Typical use cases:
//   - Dynamic content generation (e.g. marketing copy, reports)
//   - AI agent orchestration with branching logic
//   - Decision trees based on LLM outputs
//   - Automation pipelines involving prompts and external system calls
package taskengine

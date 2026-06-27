// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package trace

import (
	"fmt"
	"path/filepath"
	"strings"
)

// TraceInputError is returned when one or more trace-related CLI inputs are
// invalid. Each element in Failures is an actionable description of a single
// problem, so users can fix all issues in one pass.
type TraceInputError struct {
	Failures []string
}

func (e *TraceInputError) Error() string {
	if len(e.Failures) == 1 {
		return e.Failures[0]
	}
	lines := make([]string, 0, len(e.Failures)+1)
	lines = append(lines, fmt.Sprintf("%d trace input validation error(s):", len(e.Failures)))
	for i, f := range e.Failures {
		lines = append(lines, fmt.Sprintf("  %d. %s", i+1, f))
	}
	return strings.Join(lines, "\n")
}

// ValidateTraceInputs checks trace-related CLI flags for validity before any
// simulation or network fetch occurs.
//
// Parameters:
//   - verbosity: value of --trace-verbosity (may be empty → default normal)
//   - exportFormat: value of --format (may be empty → default text)
//   - eventFilter: value of an event-type filter (may be empty → no filter)
//   - outputPath: path supplied to --trace-output (may be empty → no export)
//
// Returns nil when all inputs are valid, or a *TraceInputError listing every
// problem found.
func ValidateTraceInputs(verbosity, exportFormat, eventFilter, outputPath string) error {
	var failures []string

	// Verbosity.
	if verbosity != "" {
		if _, err := ParseVerbosity(verbosity); err != nil {
			failures = append(failures, fmt.Sprintf(
				"invalid --trace-verbosity %q — must be one of: summary, normal, verbose\n"+
					"  Fix: use --trace-verbosity normal (default), summary (minimal), or verbose (detailed)",
				verbosity,
			))
		}
	}

	// Export format.
	if exportFormat != "" {
		normalizedFormat := strings.ToLower(strings.TrimSpace(exportFormat))
		switch normalizedFormat {
		case "text", "json", "html", "markdown", "md":
			// valid
		default:
			failures = append(failures, fmt.Sprintf(
				"invalid trace export format %q — must be one of: text, json, html, markdown\n"+
					"  Fix: use --format html (interactive), json (machine-readable), markdown (shareable), or text (CLI output)",
				exportFormat,
			))
		}
	}

	// Event filter.
	if eventFilter != "" {
		valid := false
		for _, t := range AllFilterableEventTypes() {
			if strings.EqualFold(eventFilter, t) {
				valid = true
				break
			}
		}
		if !valid {
			failures = append(failures, fmt.Sprintf(
				"invalid event filter %q — must be one of: %s\n"+
					"  Fix: choose a valid event type to filter trace output\n"+
					"  Available types: %s",
				eventFilter,
				strings.Join(AllFilterableEventTypes(), ", "),
				strings.Join(AllFilterableEventTypes(), ", "),
			))
		}
	}

	// Output path sanity: must not be a bare directory path.
	if outputPath != "" {
		if strings.HasSuffix(outputPath, "/") || strings.HasSuffix(outputPath, "\\") {
			failures = append(failures, fmt.Sprintf(
				"--trace-output %q looks like a directory path; provide a full file path\n"+
					"  Fix: specify a complete file path (e.g. ./traces/trace.html or ./output/trace.json)\n"+
					"  Example: glassbox debug --trace-output ./traces/debug-$(date +%%Y%%m%%d).html <tx-hash>",
				outputPath,
			))
		}

		// Null bytes in paths are a shell-injection risk.
		if strings.ContainsRune(outputPath, 0) {
			failures = append(failures, fmt.Sprintf(
				"--trace-output contains null bytes which are not allowed in file paths\n"+
					"  Fix: remove any null bytes from the path specification",
			))
		}

		// Use filepath.Clean to reliably detect traversal after normalisation.
		// A string-contains("..")  check would falsely flag names like "..safe"
		// or legitimate double-dot-free paths on some platforms.
		cleaned := filepath.Clean(outputPath)
		if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
			failures = append(failures, fmt.Sprintf(
				"--trace-output %q contains directory traversal sequences (..)\n"+
					"  Fix: use absolute paths or relative paths without '..' for security\n"+
					"  Example: use './output/trace.html' instead of '../output/trace.html'",
				outputPath,
			))
		}
	}

	if len(failures) > 0 {
		return &TraceInputError{Failures: failures}
	}
	return nil
}

// ValidateEventTypeField checks whether an explicitly supplied EventType value
// in an ExecutionState is a known, supported value. Unknown values are
// normalised to EventTypeOther by ClassifyEventType — calling this function
// allows callers to surface a warning when the simulator emits an unrecognised
// event type rather than silently discarding it.
//
// Returns a non-empty diagnostic string when the value is unrecognised.
func ValidateEventTypeField(eventType string) string {
	if eventType == "" {
		return "" // empty is fine; the type will be inferred
	}
	normalised := normalizeEventType(eventType)
	if normalised == EventTypeOther {
		return fmt.Sprintf(
			"unrecognised event type %q (normalised to %q); "+
				"expected one of: %s. Trace accuracy may be reduced for this step. "+
				"Check that your simulator version is compatible with this version of Glassbox",
			eventType,
			EventTypeOther,
			strings.Join(append(AllFilterableEventTypes(), EventTypeOther), ", "),
		)
	}
	return ""
}

// ValidateExecutionTrace checks an ExecutionTrace for structural correctness
// and returns a list of diagnostic messages (non-fatal unless otherwise noted).
//
// Checks:
//   - Trace is not nil.
//   - States slice is not empty (empty trace → diagnostic warning).
//   - Each state has a non-negative Step that matches its slice index.
//   - Unrecognised EventType fields are noted with their step index.
//
// This is deliberately permissive: it returns all issues at once so callers can
// choose whether to abort or merely warn.
func ValidateExecutionTrace(t *ExecutionTrace) []string {
	if t == nil {
		return []string{"execution trace is nil"}
	}

	var issues []string

	if len(t.States) == 0 {
		issues = append(issues, fmt.Sprintf(
			"execution trace for transaction %q contains no steps — "+
				"the simulator did not produce any diagnostic events. "+
				"Check that the transaction envelope is valid and the simulator binary is up-to-date",
			truncateForDiag(t.TransactionHash),
		))
		return issues // nothing further to check on an empty trace
	}

	// Per-step checks.
	for i, state := range t.States {
		if state.Step != i {
			issues = append(issues, fmt.Sprintf(
				"step index mismatch at position %d: state.Step=%d "+
					"(trace may have been modified after construction; trace accuracy may be affected)",
				i, state.Step,
			))
		}
		if diag := ValidateEventTypeField(state.EventType); diag != "" {
			issues = append(issues, fmt.Sprintf("step %d: %s", i, diag))
		}
	}

	return issues
}

// truncateForDiag trims a string for use in diagnostic messages.
func truncateForDiag(s string) string {
	if len(s) > 20 {
		return s[:17] + "..."
	}
	return s
}

// ValidateTraceExportParams validates all parameters before attempting to export a trace.
// This comprehensive check catches configuration issues before any expensive operations.
//
// Parameters:
//   - trace: the execution trace to export (must not be nil)
//   - format: export format (html, markdown, json, text)
//   - outputPath: destination file path
//   - opts: export options (comments, metadata)
//
// Returns a detailed error if validation fails, or nil if all checks pass.
func ValidateTraceExportParams(trace *ExecutionTrace, format, outputPath string, opts ExportOptions) error {
	var failures []string

	// Trace must not be nil
	if trace == nil {
		failures = append(failures, "execution trace is nil — cannot export an empty trace\n"+
			"  Fix: ensure the simulation completed successfully before attempting export\n"+
			"  Check: run the debug command without --trace-output first to verify simulation succeeds")
	}

	// Format validation
	if format == "" {
		failures = append(failures, "export format is empty — must specify one of: html, markdown, json, text\n"+
			"  Fix: provide --format html (default), markdown, json, or text")
	} else {
		normalizedFormat := strings.ToLower(strings.TrimSpace(format))
		switch normalizedFormat {
		case "html", "markdown", "md", "json", "text":
			// valid
		default:
			failures = append(failures, fmt.Sprintf(
				"unsupported export format %q — must be one of: html, markdown, json, text\n"+
					"  Fix: use a supported format\n"+
					"  Recommended: html for interactive viewing, json for CI/CD pipelines",
				format,
			))
		}
	}

	// Output path validation
	if outputPath == "" {
		failures = append(failures, "output path is empty — must specify where to write the trace\n"+
			"  Fix: provide --trace-output with a valid file path\n"+
			"  Example: --trace-output ./traces/debug-output.html")
	} else {
		// Check for invalid characters
		if strings.ContainsRune(outputPath, 0) {
			failures = append(failures, "output path contains null bytes which are not allowed\n"+
				"  Fix: remove any null bytes from the path")
		}
		
		// Check it's not a directory
		if strings.HasSuffix(outputPath, "/") || strings.HasSuffix(outputPath, "\\") {
			failures = append(failures, fmt.Sprintf(
				"output path %q appears to be a directory; must be a file path\n"+
					"  Fix: append a filename (e.g. %strace.html)",
				outputPath, outputPath,
			))
		}
	}

	// Validate trace has content
	if trace != nil && len(trace.States) == 0 {
		failures = append(failures, "execution trace contains no steps — trace export would be empty\n"+
			"  Possible causes:\n"+
			"    - Simulation did not produce any diagnostic events\n"+
			"    - Transaction envelope is invalid\n"+
			"    - Simulator version is incompatible\n"+
			"  Fix: verify the transaction executed successfully\n"+
			"  Recommended: run 'glassbox doctor' to check simulator compatibility")
	}

	// Validate export options
	if len(opts.Comments) > 100 {
		failures = append(failures, fmt.Sprintf(
			"too many comments (%d) — maximum is 100 comments per trace export\n"+
				"  Fix: reduce the number of comments or split into multiple exports",
			len(opts.Comments),
		))
	}
	
	for i, comment := range opts.Comments {
		if len(comment) > 10000 {
			failures = append(failures, fmt.Sprintf(
				"comment #%d exceeds maximum length (10000 chars) — got %d chars\n"+
					"  Fix: shorten the comment or split it into multiple comments",
				i+1, len(comment),
			))
		}
	}

	if len(failures) > 0 {
		return &TraceInputError{Failures: failures}
	}
	return nil
}

// ValidateJSONSchemaVersion validates a schema_version string as found in the
// ExportJSON envelope produced by --output-json. It rejects empty, malformed
// (not MAJOR.MINOR), or unsupported version strings with actionable messages.
//
// This is a pure-function validator suitable for use in PreRunE or any point
// where a schema version string is known before file I/O begins.
func ValidateJSONSchemaVersion(version string) error {
	if strings.TrimSpace(version) == "" {
		return &TraceInputError{Failures: []string{
			"schema_version is empty — a valid version string is required\n" +
				"  Expected format: \"MAJOR.MINOR\" (e.g. \"1.0\")\n" +
				"  Fix: use the current schema version: \"" + CurrentJSONSchemaVersion + "\"",
		}}
	}

	// Must match MAJOR.MINOR pattern (digits only, exactly two components).
	parts := strings.Split(version, ".")
	if len(parts) != 2 {
		return &TraceInputError{Failures: []string{fmt.Sprintf(
			"schema_version %q is not in MAJOR.MINOR format\n"+
				"  Expected a two-component version string (e.g. \"1.0\")\n"+
				"  Fix: use the current schema version: %q",
			version, CurrentJSONSchemaVersion,
		)}}
	}
	for _, p := range parts {
		if len(p) == 0 {
			return &TraceInputError{Failures: []string{fmt.Sprintf(
				"schema_version %q contains an empty component\n"+
					"  Fix: use a valid version such as %q",
				version, CurrentJSONSchemaVersion,
			)}}
		}
		for _, c := range p {
			if c < '0' || c > '9' {
				return &TraceInputError{Failures: []string{fmt.Sprintf(
					"schema_version %q contains non-numeric characters\n"+
						"  Expected: digits only (e.g. \"1.0\")\n"+
						"  Fix: use a valid schema version such as %q",
					version, CurrentJSONSchemaVersion,
				)}}
			}
		}
	}

	if !IsJSONSchemaVersionSupported(version) {
		return &TraceInputError{Failures: []string{fmt.Sprintf(
			"schema_version %q is not supported by this version of Glassbox\n"+
				"  Supported versions: %s\n"+
				"  Fix: re-export the trace with the current CLI, which produces schema version %q\n"+
				"  Tip: run 'glassbox trace --output-json <file> <trace-file>' to re-export",
			version,
			joinSupportedVersions(),
			CurrentJSONSchemaVersion,
		)}}
	}

	return nil
}

// joinSupportedVersions formats SupportedJSONSchemaVersions for error messages.
func joinSupportedVersions() string {
	parts := make([]string, len(SupportedJSONSchemaVersions))
	for i, v := range SupportedJSONSchemaVersions {
		parts[i] = fmt.Sprintf("%q", v)
	}
	return strings.Join(parts, ", ")
}

// ValidateTraceFormatCompatibility checks if the trace data is compatible with the target export format.
// Some formats may have specific requirements or limitations.
func ValidateTraceFormatCompatibility(trace *ExecutionTrace, format string) error {
	if trace == nil {
		return fmt.Errorf("trace is nil")
	}

	normalizedFormat := strings.ToLower(strings.TrimSpace(format))
	
	switch normalizedFormat {
	case "json":
		// JSON format requires serializable data
		for i, state := range trace.States {
			if state.ContractMetadata != nil {
				// Check for circular references or other serialization issues
				// This is a basic check; the actual JSON marshaling will catch deeper issues
				if state.ContractMetadata.Name == "" && state.ContractMetadata.Version == "" {
					// This might indicate incomplete metadata that could cause serialization issues
					// But it's not a hard error, just a warning
				}
			}
			if state.Step != i {
				return fmt.Errorf("trace step mismatch at position %d: expected step %d but got %d — trace may be corrupted", i, i, state.Step)
			}
		}
		
	case "html":
		// HTML format has special character escaping requirements
		// Check for extremely long strings that might cause browser issues
		for i, state := range trace.States {
			argStr := fmt.Sprintf("%v", state.Arguments)
			if len(argStr) > 50000 {
				return fmt.Errorf("step %d has very large arguments (%d chars) that may cause browser rendering issues in HTML format — consider using JSON format instead", i, len(argStr))
			}
		}
		
	case "markdown", "md":
		// Markdown format works well with most data but very long lines can be problematic
		// This is a soft check
		
	case "text":
		// Plain text format is the most permissive
	}
	
	return nil
}

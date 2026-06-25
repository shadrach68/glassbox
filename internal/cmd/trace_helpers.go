// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"
)

// humanFileSize converts a byte count to a compact, human-readable string
// (e.g. "1.5 MB"). Used when reporting exported file sizes on the CLI.
func humanFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// traceExportedFileSize returns a parenthesised human-readable size string for
// the file at path, or an empty string when the file cannot be stat'd.
// Intended for use in success messages: "Trace exported to: trace.html (42 KB)".
func traceExportedFileSize(path string) string {
	if path == "" {
		return ""
	}
	info, err := os.Stat(path)
	if err != nil {
		return ""
	}
	return " (" + humanFileSize(info.Size()) + ")"
}

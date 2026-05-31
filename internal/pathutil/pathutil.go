// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

// Package pathutil provides cross-platform path normalization utilities for
// source mapping, configuration loading, and CLI input parsing. All functions
// handle Windows drive letters (e.g. C:\foo) and mixed separator styles.
package pathutil

import (
	"path/filepath"
	"strings"
)

// Normalize converts sep to the OS-native separator and cleans the path.
// It is safe to call on both POSIX and Windows paths regardless of the
// current platform.
func Normalize(p string) string {
	// Convert any forward or backward slashes to filepath.Separator.
	p = filepath.FromSlash(strings.ReplaceAll(p, "\\", "/"))
	return filepath.Clean(p)
}

// ToSlash converts all path separators in p to forward slashes and cleans the
// result. Use this when building URLs or any platform-neutral representation
// such as GitHub blob links or source-map keys.
func ToSlash(p string) string {
	return filepath.ToSlash(filepath.Clean(p))
}

// Join joins path elements using the OS-native separator, normalizing each
// element first so that Windows-style components work correctly on all
// platforms.
func Join(parts ...string) string {
	normalized := make([]string, len(parts))
	for i, p := range parts {
		normalized[i] = filepath.FromSlash(strings.ReplaceAll(p, "\\", "/"))
	}
	return filepath.Join(normalized...)
}

// IsWindowsAbs returns true when p looks like an absolute Windows path,
// i.e. it starts with a drive letter (e.g. "C:\" or "C:/").
// This is safe to call on any platform.
func IsWindowsAbs(p string) bool {
	if len(p) < 3 {
		return false
	}
	drive := p[0]
	return ((drive >= 'A' && drive <= 'Z') || (drive >= 'a' && drive <= 'z')) &&
		p[1] == ':' && (p[2] == '/' || p[2] == '\\')
}

// NormalizeForGitHub converts p to a relative forward-slash path suitable for
// embedding in a GitHub blob URL. If p is absolute (POSIX or Windows), it is
// returned as-is after separator normalisation; callers are expected to make it
// relative to the repo root with filepath.Rel before building the URL.
func NormalizeForGitHub(p string) string {
	// Replace backslashes with forward slashes uniformly.
	p = strings.ReplaceAll(p, "\\", "/")
	// Strip any Windows drive prefix so the path is host-root-relative.
	if IsWindowsAbs(p) {
		p = p[2:] // strip "C:"
	}
	// Collapse multiple slashes and resolve "." / "..".
	p = filepath.ToSlash(filepath.Clean(filepath.FromSlash(p)))
	// Ensure no leading slash remains.
	return strings.TrimPrefix(p, "/")
}

// RelToSlash is a convenience wrapper around filepath.Rel that always returns
// a forward-slash path, making it safe to use in GitHub URLs regardless of the
// platform on which the CLI is running.
func RelToSlash(basepath, targpath string) (string, error) {
	rel, err := filepath.Rel(Normalize(basepath), Normalize(targpath))
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(rel), nil
}

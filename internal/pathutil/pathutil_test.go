// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package pathutil

import (
	"runtime"
	"strings"
	"testing"
)

func TestIsWindowsAbs(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{`C:\Users\foo`, true},
		{`C:/Users/foo`, true},
		{`c:\foo`, true},
		{`/usr/local/bin`, false},
		{`relative/path`, false},
		{`D:\`, true},
		{``, false},
		{`C:`, false},
	}
	for _, tt := range tests {
		got := IsWindowsAbs(tt.path)
		if got != tt.want {
			t.Errorf("IsWindowsAbs(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestNormalizeForGitHub(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{`contracts/token/src/lib.rs`, `contracts/token/src/lib.rs`},
		{`contracts\token\src\lib.rs`, `contracts/token/src/lib.rs`},
		{`C:\contracts\token\src\lib.rs`, `contracts/token/src/lib.rs`},
		{`C:/contracts/token/src/lib.rs`, `contracts/token/src/lib.rs`},
		{`/usr/local/src/lib.rs`, `usr/local/src/lib.rs`},
	}
	for _, tt := range tests {
		got := NormalizeForGitHub(tt.path)
		if got != tt.want {
			t.Errorf("NormalizeForGitHub(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestToSlash(t *testing.T) {
	got := ToSlash(`foo/bar/baz`)
	if got != "foo/bar/baz" {
		t.Errorf("ToSlash = %q, want %q", got, "foo/bar/baz")
	}
	// Always forward slashes in output.
	if strings.Contains(got, "\\") {
		t.Errorf("ToSlash result contains backslashes: %q", got)
	}
}

func TestJoin(t *testing.T) {
	got := Join("foo", "bar", "baz.rs")
	if runtime.GOOS == "windows" {
		if !strings.Contains(got, "foo") || !strings.Contains(got, "baz.rs") {
			t.Errorf("Join produced unexpected result on Windows: %q", got)
		}
	} else {
		want := "foo/bar/baz.rs"
		if got != want {
			t.Errorf("Join = %q, want %q", got, want)
		}
	}
}

func TestNormalize(t *testing.T) {
	// Mixed separators collapse correctly.
	in := `contracts/token\src/lib.rs`
	got := Normalize(in)
	if strings.Contains(got, "/") && runtime.GOOS == "windows" {
		t.Errorf("Normalize on Windows left forward slashes: %q", got)
	}
	if strings.Contains(got, "\\") && runtime.GOOS != "windows" {
		t.Errorf("Normalize on POSIX left backslashes: %q", got)
	}
}

func TestRelToSlash(t *testing.T) {
	rel, err := RelToSlash("/repo/root", "/repo/root/contracts/token/src/lib.rs")
	if err != nil {
		t.Fatalf("RelToSlash error: %v", err)
	}
	if rel != "contracts/token/src/lib.rs" {
		t.Errorf("RelToSlash = %q, want %q", rel, "contracts/token/src/lib.rs")
	}
}

// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package trace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExportExecutionTrace_HTMLAndMarkdown(t *testing.T) {
	trace := NewExecutionTrace("test-tx-hash", 10)
	trace.StartTime = time.Date(2026, time.January, 2, 15, 4, 5, 0, time.UTC)
	trace.EndTime = trace.StartTime.Add(5 * time.Minute)

	trace.AddState(ExecutionState{
		Operation:  "contract_call",
		EventType:  "contract_call",
		ContractID: "C123",
		Function:   "transfer",
		Arguments:  []interface{}{"100", "XLM"},
		ReturnValue: "ok",
		SourceFile: "src/contract.rs",
		SourceLine: 42,
		GitHubLink: "https://github.com/example/repo/blob/main/src/contract.rs#L42",
	})
	trace.AddState(ExecutionState{
		Operation: "host_function",
		Error:     "insufficient balance",
	})

	tmpDir := t.TempDir()
	htmlPath := filepath.Join(tmpDir, "trace-export.html")
	if err := ExportExecutionTrace(trace, "html", htmlPath); err != nil {
		t.Fatalf("ExportExecutionTrace(html) failed: %v", err)
	}
	data, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatalf("failed to read exported html file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "Glassbox Trace Export") {
		t.Fatalf("exported html missing expected header")
	}
	if !strings.Contains(content, "contract_call") {
		t.Fatalf("exported html missing step operation")
	}

	mdPath := filepath.Join(tmpDir, "trace-export.md")
	if err := ExportExecutionTrace(trace, "markdown", mdPath); err != nil {
		t.Fatalf("ExportExecutionTrace(markdown) failed: %v", err)
	}
	data, err = os.ReadFile(mdPath)
	if err != nil {
		t.Fatalf("failed to read exported markdown file: %v", err)
	}
	content = string(data)
	if !strings.Contains(content, "# Glassbox Trace Export") {
		t.Fatalf("exported markdown missing expected header")
	}
	if !strings.Contains(content, "transfer") {
		t.Fatalf("exported markdown missing expected function name")
	}

	txtPath := filepath.Join(tmpDir, "trace-export.txt")
	if err := ExportExecutionTrace(trace, "text", txtPath); err != nil {
		t.Fatalf("ExportExecutionTrace(text) failed: %v", err)
	}
	data, err = os.ReadFile(txtPath)
	if err != nil {
		t.Fatalf("failed to read exported text file: %v", err)
	}
	content = string(data)
	if !strings.Contains(content, "Glassbox Trace Export") {
		t.Fatalf("exported text missing expected header")
	}
	if !strings.Contains(content, "  Contract:  C123") {
		t.Fatalf("exported text missing indented contract field")
	}
	if !strings.Contains(content, "  Source:") {
		t.Fatalf("exported text missing source reference section")
	}
}

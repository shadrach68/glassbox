// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/dotandev/glassbox/internal/config"
	"github.com/dotandev/glassbox/internal/errors"
	"github.com/dotandev/glassbox/internal/signer"
	"github.com/dotandev/glassbox/internal/version"
)

const operationAuditLogVersion = "1.0.0"

var sensitiveFlagPattern = strings.NewReplacer(
	"token", "token",
	"secret", "secret",
	"password", "password",
	"private", "private",
	"key", "key",
	"pin", "pin",
	"passphrase", "passphrase",
)

// OperationAuditEntry is a stable key/value pair used for signed audit records.
type OperationAuditEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// OperationAuditCommand captures the CLI command name and arguments.
type OperationAuditCommand struct {
	Path             string   `json:"path"`
	Args             []string `json:"args"`
	WorkingDirectory string   `json:"working_directory"`
	CliVersion       string   `json:"cli_version"`
}

// OperationAuditLog is the unsigned operation audit record.
type OperationAuditLog struct {
	Version     string                  `json:"version"`
	Timestamp   time.Time               `json:"timestamp"`
	Command     OperationAuditCommand   `json:"command"`
	Config      []OperationAuditEntry   `json:"config,omitempty"`
	Environment []OperationAuditEntry   `json:"environment,omitempty"`
	Metadata    []OperationAuditEntry   `json:"metadata,omitempty"`
	Success     bool                    `json:"success"`
	Error       string                  `json:"error,omitempty"`
}

// SignedOperationAuditLog is the signed operation audit payload stored to disk.
type SignedOperationAuditLog struct {
	Version   string            `json:"version"`
	Timestamp time.Time         `json:"timestamp"`
	Payload   OperationAuditLog `json:"payload"`
	TraceHash string            `json:"trace_hash"`
	Signature string            `json:"signature"`
	PublicKey string            `json:"public_key"`
	Provider  string            `json:"provider"`
}

func writeOperationAuditLog(rawArgs []string, execErr error) error {
	record, err := buildOperationAuditRecord(rawArgs, execErr)
	if err != nil {
		return err
	}

	signedLog, err := signOperationAuditRecord(record)
	if err != nil {
		return err
	}

	output, err := json.MarshalIndent(signedLog, "", "  ")
	if err != nil {
		return errors.WrapMarshalFailed(err)
	}

	path := AuditLogPathFlag
	if path == "" {
		path = defaultAuditLogPath()
	}

	// Validate the audit log output path before creating directories or writing.
	// This catches null bytes, traversal sequences, and existing-directory targets
	// that would otherwise reach os.MkdirAll / os.WriteFile unvalidated.
	if _, err := ValidateOutputPath("audit-log", path); err != nil {
		return errors.WrapValidationError(fmt.Sprintf("invalid --audit-log path: %v", err))
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return errors.WrapValidationError(fmt.Sprintf("failed to create audit log directory: %v", err))
	}

	if err := os.WriteFile(path, output, 0o600); err != nil {
		return errors.WrapValidationError(fmt.Sprintf("failed to write audit log: %v", err))
	}

	fmt.Fprintf(os.Stderr, "Operation audit log saved to %s\n", path)
	return nil
}

func buildOperationAuditRecord(rawArgs []string, execErr error) (*OperationAuditLog, error) {
	wd, _ := os.Getwd()

	command := OperationAuditCommand{
		Path:             rawArgs[0],
		Args:             sanitizeArgs(rawArgs[1:]),
		WorkingDirectory: wd,
		CliVersion:       version.Version,
	}

	configEntries, err := buildConfigEntries()
	if err != nil {
		return nil, err
	}

	return &OperationAuditLog{
		Version:     operationAuditLogVersion,
		Timestamp:   time.Now().UTC(),
		Command:     command,
		Config:      configEntries,
		Environment: buildEnvironmentEntries(),
		Metadata:    parseMetadataEntries(AuditLogMetadata),
		Success:     execErr == nil,
		Error:       sanitizeError(execErr),
	}, nil
}

func defaultAuditLogPath() string {
	timestamp := time.Now().UTC().Format("20060102-150405")
	return fmt.Sprintf("glassbox-operation-audit-%s.json", timestamp)
}

func buildConfigEntries() ([]OperationAuditEntry, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil
	}

	raw, err := json.Marshal(cfg)
	if err != nil {
		return nil, errors.WrapMarshalFailed(err)
	}

	var values map[string]interface{}
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, errors.WrapUnmarshalFailed(err, "config")
	}

	return sortedEntries(values), nil
}

func buildEnvironmentEntries() []OperationAuditEntry {
	entries := []OperationAuditEntry{
		{Key: "os", Value: runtime.GOOS},
		{Key: "arch", Value: runtime.GOARCH},
		{Key: "shell", Value: os.Getenv("SHELL")},
		{Key: "term", Value: os.Getenv("TERM")},
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Key < entries[j].Key
	})
	return entries
}

func parseMetadataEntries(source []string) []OperationAuditEntry {
	entries := make([]OperationAuditEntry, 0, len(source))
	for _, entry := range source {
		if strings.TrimSpace(entry) == "" {
			continue
		}
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			continue
		}
		entries = append(entries, OperationAuditEntry{Key: strings.TrimSpace(parts[0]), Value: strings.TrimSpace(parts[1])})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Key < entries[j].Key })
	return entries
}

func sortedEntries(values map[string]interface{}) []OperationAuditEntry {
	entries := make([]OperationAuditEntry, 0, len(values))
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		val := values[key]
		entries = append(entries, OperationAuditEntry{Key: key, Value: sanitizeValue(key, val)})
	}
	return entries
}

func sanitizeValue(key string, value interface{}) string {
	if isSensitiveKey(key) {
		return "REDACTED"
	}
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return sanitizeArgValue(key, v)
	case float64, bool, int, int64, uint64:
		return fmt.Sprintf("%v", v)
	default:
		if jsonBytes, err := json.Marshal(v); err == nil {
			return string(jsonBytes)
		}
		return fmt.Sprintf("%v", v)
	}
}

func sanitizeArgValue(key, value string) string {
	if isSensitiveKey(key) {
		return "REDACTED"
	}
	if len(value) > 0 && isLikelySecret(value) {
		return "REDACTED"
	}
	return value
}

func isSensitiveKey(key string) bool {
	lower := strings.ToLower(key)
	return strings.Contains(lower, "token") || strings.Contains(lower, "secret") || strings.Contains(lower, "password") || strings.Contains(lower, "private") || strings.Contains(lower, "key") || strings.Contains(lower, "pin") || strings.Contains(lower, "passphrase")
}

func isLikelySecret(value string) bool {
	return len(value) >= 16 && strings.ContainsAny(value, "0123456789abcdefABCDEF")
}

func sanitizeError(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func sanitizeArgs(rawArgs []string) []string {
	result := make([]string, 0, len(rawArgs))
	for i := 0; i < len(rawArgs); i++ {
		arg := rawArgs[i]
		if strings.HasPrefix(arg, "--") {
			name, _, hasValue := splitFlag(arg)
			if isSensitiveKey(name) {
				if hasValue {
					result = append(result, fmt.Sprintf("%s=REDACTED", name))
				} else {
					result = append(result, name)
					if i+1 < len(rawArgs) {
						i++
						result = append(result, "REDACTED")
					}
				}
				continue
			}
			if hasValue {
				result = append(result, arg)
				continue
			}
		}
		result = append(result, arg)
	}
	return result
}

func splitFlag(flag string) (string, string, bool) {
	if idx := strings.Index(flag, "="); idx != -1 {
		return flag[:idx], flag[idx+1:], true
	}
	return flag, "", false
}

func signOperationAuditRecord(record *OperationAuditLog) (*SignedOperationAuditLog, error) {
	providerName, cfg := resolveAuditSignerProviderAndConfig()

	signerImpl, err := signer.DefaultRegistry.CreateSigner(providerName, cfg)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closer, ok := signerImpl.(interface{ Close() error }); ok {
			_ = closer.Close()
		}
	}()

	payloadBytes, err := json.Marshal(record)
	if err != nil {
		return nil, errors.WrapMarshalFailed(err)
	}

	hash := sha256.Sum256(payloadBytes)
	sig, err := signerImpl.Sign(hash[:])
	if err != nil {
		return nil, errors.WrapValidationError(fmt.Sprintf("failed to sign operation audit record: %v", err))
	}

	pubKey, err := signerImpl.PublicKey()
	if err != nil {
		return nil, errors.WrapValidationError(fmt.Sprintf("failed to retrieve public key: %v", err))
	}

	return &SignedOperationAuditLog{
		Version:   operationAuditLogVersion,
		Timestamp: time.Now().UTC(),
		Payload:   *record,
		TraceHash: hex.EncodeToString(hash[:]),
		Signature: hex.EncodeToString(sig),
		PublicKey: hex.EncodeToString(pubKey),
		Provider:  providerName,
	}, nil
}

func resolveAuditSignerProviderAndConfig() (string, signer.ProviderConfig) {
	cfg := signer.ProviderConfig{
		SoftwareKeyPEM:   AuditLogSoftwareKey,
		PKCS11ModulePath: AuditLogPKCS11Module,
		PKCS11PIN:        AuditLogPKCS11PIN,
		PKCS11TokenLabel: AuditLogPKCS11TokenLabel,
		PKCS11KeyLabel:   AuditLogPKCS11KeyLabel,
		PKCS11KeyIDHex:   AuditLogPKCS11KeyIDHex,
	}

	name := AuditLogProviderFlag
	if name == "" {
		name = os.Getenv("GLASSBOX_AUDIT_SIGNING_PROVIDER")
	}
	if name == "" {
		name = os.Getenv("GLASSBOX_SIGNING_PROVIDER")
	}
	if name == "" {
		name = os.Getenv("GLASSBOX_SIGNER_TYPE")
	}
	if name == "" {
		name = "software"
	}

	return strings.ToLower(name), cfg
}

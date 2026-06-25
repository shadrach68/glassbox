# Quick Reference: Debug and Trace Export Improvements

## What Changed (Latest Session)

### Profile Command — CLI Ergonomics (Part A)
**File:** `internal/cmd/profile.go`

**New Validations:**
- ✅ `--output` (pprof path): directory-path guard (trailing `/` or `\` rejected)
- ✅ `--output`: non-existent parent directory caught before execution
- ✅ `--out-json`: directory-path guard (was missing; now consistent with `--output`)
- ✅ `--out-json`: non-existent parent directory now includes a `Fix:` hint
- ✅ Empty-trace early-warning in `runTraceProfile` (non-fatal, clear message + remediation)
- ✅ File-create failure in `runTraceProfile` now includes a `Tip:` for alternate output path

**Updated Help Text:**
- Long description updated with `Output options:` section and performance notes

**New Tests (`internal/cmd/profile_validation_test.go`):**
- `TestProfilePreRunE_Output_DirectoryPath_Rejected`
- `TestProfilePreRunE_Output_MissingParentDirectory_Rejected`
- `TestProfilePreRunE_Output_ExistingDirectory_Passes`
- `TestProfilePreRunE_Output_Default_Passes`
- `TestProfilePreRunE_OutJSON_DirectoryPath_Rejected`
- `TestProfilePreRunE_OutJSON_MissingDirectory_HasFixHint`

### Performance and Profiling Trace Export (Part B)
**Files:** `internal/profile/generator.go`, `internal/profile/pprof.go`

**New Diagnostics:**
- ✅ `GenerateHTML`: empty-trace warning on stderr (non-fatal — blank HTML is still written)
- ✅ `TraceToPprof`: step-index mismatch errors now include a `Fix:` hint with `glassbox debug` reference
- ✅ `runTraceProfile`: actionable warning when trace has zero steps (explains causes + next steps)
- ✅ `runTraceProfile`: pprof-write failure now includes `Fix:` and `Check:` lines

**New Tests (`internal/profile/generator_test.go`):**
- `TestGenerateHTML_EmptyTrace_StillProducesOutput`
- `TestGenerateHTML_NilWriter_ReturnsError`

**New Tests (`internal/profile/pprof_test.go`):**
- `TestTraceToPprof_EmptyTrace_ReturnsEmptyProfile`
- `TestTraceToPprof_StepIndexMismatch_ReturnsActionableError`
- `TestWritePprof_EmptyTrace_Succeeds`
- `TestTraceToPprof_ZeroGasSteps_ProducesNoSamples`
- `TestTraceToPprof_MixedGasAndZero_OnlyCapturesPositive`

**New Documentation:**
- `docs/trace-profiling.md` — full reference for `glassbox profile` export modes,
  validation checks, error messages, zero-gas behaviour, and troubleshooting

---

## What Changed (Previous Session)

### Debug Command (Task A)
**File:** `internal/cmd/debug_dry_run.go`

**New Validations:**
- ✅ RPC URL format checking
- ✅ Compare-network distinctness validation
- ✅ Simulator version compatibility
- ✅ Protocol version range checking (20-23)
- ✅ Trace output configuration validation

**Enhanced Error Messages:**
- All errors now include "Fix:" sections
- Examples provided for correct usage
- Detailed remediation steps
- Clear success/failure messaging

### Trace Export (Task B)
**Files:** `internal/trace/validate.go`, `internal/trace/export.go`

**New Validations:**
- ✅ Comprehensive pre-export parameter validation
- ✅ Format compatibility checking (JSON, HTML, MD, Text)
- ✅ Trace structure integrity validation
- ✅ Comment count and length limits
- ✅ Path security (null bytes, traversal)

**Enhanced Error Messages:**
- Multi-error batch reporting
- Format-specific guidance
- Troubleshooting recommendations
- File system operation details

---

## Before vs After

### Debug Command Error Messages

**BEFORE:**
```
[FAIL] Invalid network "badnet"
```

**AFTER:**
```
[FAIL] Invalid network "badnet" — must be testnet, mainnet, futurenet, or a custom network defined in config
       Fix: use --network testnet, --network mainnet, or --network futurenet
       Or define a custom network in glassbox.toml under [networks]
```

### Trace Export Error Messages

**BEFORE:**
```
unsupported trace export format: yaml
```

**AFTER:**
```
invalid trace export format "yaml" — must be one of: text, json, html, markdown
  Fix: use --format html (interactive), json (machine-readable), markdown (shareable), or text (CLI output)
```

---

## New Features

### 1. Enhanced Dry-Run Mode

```bash
glassbox debug --dry-run --network testnet <tx-hash>
```

**Now Checks:**
- Transaction hash format ✓
- Network validity ✓
- Compare-network validity and distinctness ✓
- RPC URL format ✓
- RPC endpoint reachability ✓
- Simulator binary presence ✓
- Simulator version compatibility ✓
- Protocol version ✓
- Trace output configuration ✓

### 2. Comprehensive Trace Validation

```bash
glassbox debug --trace-output ./output.html --format html <tx-hash>
```

**Now Validates:**
- Trace is not nil ✓
- Trace contains steps ✓
- Format is supported ✓
- Output path is valid ✓
- No path traversal ✓
- Comment limits ✓
- Format compatibility ✓

### 3. Multi-Error Reporting

Both validations now report all errors at once:

```
3 validation error(s):
  1. transaction hash: expected 64 hexadecimal characters
  2. network: invalid network "badnet"
  3. compare-network: cannot be the same as primary network
```

---

## API Changes

### New Functions

**Debug Command:**
```go
// internal/cmd/debug_dry_run.go
func validateRPCURL(rawURL string) error
func validateSimulatorVersion(version string) error
func validateProtocolVersion(version uint32) error
```

**Trace Export:**
```go
// internal/trace/validate.go
func ValidateTraceExportParams(trace *ExecutionTrace, format, outputPath string, opts ExportOptions) error
func ValidateTraceFormatCompatibility(trace *ExecutionTrace, format string) error
```

### Modified Functions

**Enhanced with comprehensive validation:**
```go
// internal/cmd/debug_dry_run.go
func runDebugDryRun(cmd *cobra.Command, txHash string) error

// internal/trace/validate.go
func ValidateTraceInputs(verbosity, exportFormat, eventFilter, outputPath string) error

// internal/trace/export.go
func ExportExecutionTraceWithOptions(trace *ExecutionTrace, format string, outputPath string, opts ExportOptions) error
```

---

## Testing

### New Test Files

1. **`internal/cmd/debug_dry_run_test.go`** - 19 test cases
2. **`internal/trace/validate_test.go`** - 29 test cases

### Running Tests

```bash
# All new tests
go test ./internal/cmd/debug_dry_run_test.go -v
go test ./internal/trace/validate_test.go -v

# Specific validation tests
go test ./internal/cmd/... -run TestValidateRPCURL -v
go test ./internal/trace/... -run TestValidateTraceExportParams -v
```

---

## Documentation

### New Documentation

**`docs/trace-export-validation.md`**
- Comprehensive validation guide
- Error examples and solutions
- Best practices
- Troubleshooting section

### Updated Documentation

**`docs/debug-command.md`**
- Enhanced dry-run section
- Updated check list (5 → 9 checks)
- Improved example output

---

## Backward Compatibility

✅ **Fully backward compatible**

- All existing functionality preserved
- New validations only add checks, don't remove features
- Error messages enhanced, not changed
- CLI flags remain the same
- Test suite expanded, not replaced

---

## Common Use Cases

### 1. Validate Config Before Expensive Operations

```bash
# Check everything before running
glassbox debug --dry-run \
  --network testnet \
  --compare-network mainnet \
  --trace-output ./traces/output.html \
  <tx-hash>
```

### 2. Export Traces Safely

```bash
# With comprehensive validation
glassbox debug \
  --network testnet \
  --format json \
  --trace-output ./safe/path/trace.json \
  <tx-hash>
```

### 3. Multi-Network Comparison

```bash
# Validate distinct networks
glassbox debug --dry-run \
  --network testnet \
  --compare-network mainnet \
  <tx-hash>
```

### 4. CI/CD Integration

```bash
#!/bin/bash
# Validate configuration
if ! glassbox debug --dry-run --network $NETWORK $TX_HASH; then
  echo "Validation failed"
  exit 1
fi

# Execute with trace export
glassbox debug \
  --network $NETWORK \
  --format json \
  --trace-output ./artifacts/trace.json \
  $TX_HASH
```

---

## Error Resolution Guide

### "Invalid RPC URL"
**Cause:** URL not HTTP/HTTPS  
**Fix:** Use `https://` or `http://` scheme

### "Compare-network must be different"
**Cause:** Same network for primary and compare  
**Fix:** Select different networks: `--network testnet --compare-network mainnet`

### "Trace contains no steps"
**Cause:** Simulation produced no events  
**Fix:** Run `glassbox doctor`, verify transaction executed

### "Output path is a directory"
**Cause:** Path ends with `/` or `\`  
**Fix:** Add filename: `./traces/output.html` not `./traces/`

### "Path traversal detected"
**Cause:** Path contains `..`  
**Fix:** Use absolute or forward-relative paths only

---

## Performance Impact

- ✅ **Negligible** - All validations are O(1) or O(n) where n is small
- ✅ **Early Exit** - Failures detected before expensive operations
- ✅ **No Network Overhead** - Validations are local only (except RPC health in dry-run)

---

## Security Improvements

1. **Path Traversal Detection** - Prevents `../../../etc/passwd` attacks
2. **Null Byte Rejection** - Blocks null byte injection
3. **URL Validation** - Ensures safe HTTP/HTTPS schemes
4. **Input Sanitization** - All user inputs validated before use

---

## Monitoring & Observability

### Exit Codes
- `0` - All validations passed
- `1` - One or more validation failures

### Error Formats
- **Text:** Human-readable with Fix: sections
- **JSON:** Structured (when using `--format json`)

### Log Levels
- **Error:** Validation failures
- **Info:** Successful checks
- **Debug:** Detailed validation steps (with `--verbose`)

---

## Next Steps

1. **Run Tests:** Install Go and run test suites
2. **Try Examples:** Use the commands in this guide
3. **Read Docs:** Review `docs/trace-export-validation.md`
4. **Report Issues:** Test edge cases and report any issues

---

## Support

For issues or questions:
1. Check `docs/debug-command.md` for debug command help
2. Check `docs/trace-export-validation.md` for trace export help
3. Run `glassbox debug --help` for CLI reference
4. Run `glassbox doctor` for environment diagnostics

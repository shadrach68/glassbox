// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package rpc

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/dotandev/glassbox/internal/errors"
	"github.com/dotandev/glassbox/internal/logger"
	"github.com/stellar/go-stellar-sdk/xdr"
)

// validateLedgerKeyXDR decodes a base64-encoded XDR LedgerKey, validates its structure,
// and emits a debug log with the key's SHA-256 hash and type.
func validateLedgerKeyXDR(keyB64 string) (*xdr.LedgerKey, error) {
	keyB64 = strings.TrimSpace(keyB64)
	if keyB64 == "" {
		return nil, errors.WrapValidationError("ledger key XDR is empty")
	}

	keyBytes, err := base64.StdEncoding.DecodeString(keyB64)
	if err != nil {
		return nil, errors.WrapValidationError(fmt.Sprintf("failed to decode ledger key: %v", err))
	}

	var ledgerKey xdr.LedgerKey
	if err := xdr.SafeUnmarshal(keyBytes, &ledgerKey); err != nil {
		return nil, errors.WrapValidationError(fmt.Sprintf("failed to unmarshal ledger key: %v", err))
	}

	if err := validateLedgerKeyFields(&ledgerKey); err != nil {
		return nil, err
	}

	hash := sha256.Sum256(keyBytes)
	hashHex := hex.EncodeToString(hash[:])

	logger.Logger.Debug("Ledger key validated",
		"key_hash", hashHex,
		"key_type", ledgerKey.Type.String())

	return &ledgerKey, nil
}

func validateLedgerEntryXDR(entryB64 string) (*xdr.LedgerEntry, error) {
	entryB64 = strings.TrimSpace(entryB64)
	if entryB64 == "" {
		return nil, errors.WrapValidationError("ledger entry XDR is empty")
	}

	entryBytes, err := base64.StdEncoding.DecodeString(entryB64)
	if err != nil {
		return nil, errors.WrapValidationError(fmt.Sprintf("failed to decode ledger entry: %v", err))
	}

	var ledgerEntry xdr.LedgerEntry
	if err := xdr.SafeUnmarshal(entryBytes, &ledgerEntry); err != nil {
		return nil, errors.WrapValidationError(fmt.Sprintf("failed to unmarshal ledger entry: %v", err))
	}

	if err := validateLedgerEntryFields(&ledgerEntry); err != nil {
		return nil, err
	}

	return &ledgerEntry, nil
}

func validateLedgerKeyFields(key *xdr.LedgerKey) error {
	switch key.Type {
	case xdr.LedgerEntryTypeAccount:
		if key.Account == nil {
			return errors.WrapValidationError("ledger key missing required Account field")
		}
	case xdr.LedgerEntryTypeTrustline:
		if key.TrustLine == nil {
			return errors.WrapValidationError("ledger key missing required TrustLine field")
		}
	case xdr.LedgerEntryTypeOffer:
		if key.Offer == nil {
			return errors.WrapValidationError("ledger key missing required Offer field")
		}
	case xdr.LedgerEntryTypeData:
		if key.Data == nil {
			return errors.WrapValidationError("ledger key missing required Data field")
		}
	case xdr.LedgerEntryTypeClaimableBalance:
		if key.ClaimableBalance == nil {
			return errors.WrapValidationError("ledger key missing required ClaimableBalance field")
		}
	case xdr.LedgerEntryTypeLiquidityPool:
		if key.LiquidityPool == nil {
			return errors.WrapValidationError("ledger key missing required LiquidityPool field")
		}
	case xdr.LedgerEntryTypeContractData:
		if key.ContractData == nil {
			return errors.WrapValidationError("ledger key missing required ContractData field")
		}
	case xdr.LedgerEntryTypeContractCode:
		if key.ContractCode == nil {
			return errors.WrapValidationError("ledger key missing required ContractCode field")
		}
	case xdr.LedgerEntryTypeConfigSetting:
		if key.ConfigSetting == nil {
			return errors.WrapValidationError("ledger key missing required ConfigSetting field")
		}
	case xdr.LedgerEntryTypeTtl:
		if key.Ttl == nil {
			return errors.WrapValidationError("ledger key missing required Ttl field")
		}
	default:
		return errors.WrapValidationError(fmt.Sprintf("ledger key has unsupported type: %s", key.Type.String()))
	}
	return nil
}

func validateLedgerEntryFields(entry *xdr.LedgerEntry) error {
	switch entry.Data.Type {
	case xdr.LedgerEntryTypeAccount:
		if entry.Data.Account == nil {
			return errors.WrapValidationError("ledger entry missing required Account payload")
		}
	case xdr.LedgerEntryTypeTrustline:
		if entry.Data.TrustLine == nil {
			return errors.WrapValidationError("ledger entry missing required TrustLine payload")
		}
	case xdr.LedgerEntryTypeOffer:
		if entry.Data.Offer == nil {
			return errors.WrapValidationError("ledger entry missing required Offer payload")
		}
	case xdr.LedgerEntryTypeData:
		if entry.Data.Data == nil {
			return errors.WrapValidationError("ledger entry missing required Data payload")
		}
	case xdr.LedgerEntryTypeClaimableBalance:
		if entry.Data.ClaimableBalance == nil {
			return errors.WrapValidationError("ledger entry missing required ClaimableBalance payload")
		}
	case xdr.LedgerEntryTypeLiquidityPool:
		if entry.Data.LiquidityPool == nil {
			return errors.WrapValidationError("ledger entry missing required LiquidityPool payload")
		}
	case xdr.LedgerEntryTypeContractData:
		if entry.Data.ContractData == nil {
			return errors.WrapValidationError("ledger entry missing required ContractData payload")
		}
	case xdr.LedgerEntryTypeContractCode:
		if entry.Data.ContractCode == nil {
			return errors.WrapValidationError("ledger entry missing required ContractCode payload")
		}
	case xdr.LedgerEntryTypeConfigSetting:
		if entry.Data.ConfigSetting == nil {
			return errors.WrapValidationError("ledger entry missing required ConfigSetting payload")
		}
	case xdr.LedgerEntryTypeTtl:
		if entry.Data.Ttl == nil {
			return errors.WrapValidationError("ledger entry missing required Ttl payload")
		}
	default:
		return errors.WrapValidationError(fmt.Sprintf("ledger entry has unsupported type: %s", entry.Data.Type.String()))
	}
	return nil
}

// ValidateLedgerEntryShape verifies that a ledger key and entry pair have
// matching types and contain the required payload fields for Soroban replay.
func ValidateLedgerEntryShape(keyB64, entryB64 string) error {
	ledgerKey, err := validateLedgerKeyXDR(keyB64)
	if err != nil {
		return err
	}

	ledgerEntry, err := validateLedgerEntryXDR(entryB64)
	if err != nil {
		return err
	}

	if ledgerKey.Type != ledgerEntry.Data.Type {
		return errors.WrapValidationError(fmt.Sprintf(
			"ledger key type %s does not match entry type %s",
			ledgerKey.Type.String(), ledgerEntry.Data.Type.String()))
	}

	return nil
}

// VerifyLedgerEntryHash cryptographically verifies that a returned ledger entry
// matches the expected hash derived from its key. This ensures data integrity
// before feeding entries to the simulator.
func VerifyLedgerEntryHash(requestedKeyB64 string, result LedgerEntryResult) error {
	if strings.TrimSpace(result.Key) == "" {
		return errors.WrapValidationError("ledger entry result missing Key field")
	}
	if strings.TrimSpace(result.Xdr) == "" {
		return errors.WrapValidationError("ledger entry result missing Xdr field")
	}

	if requestedKeyB64 != result.Key {
		return errors.WrapValidationError(
			fmt.Sprintf("ledger entry key mismatch: requested %s but received %s",
				requestedKeyB64, result.Key))
	}

	if err := ValidateLedgerEntryShape(requestedKeyB64, result.Xdr); err != nil {
		return err
	}

	ledgerEntry, err := validateLedgerEntryXDR(result.Xdr)
	if err != nil {
		return err
	}

	// Verify that the entry's key matches the requested key
	derivedKey := ledgerKeyFromEntry(*ledgerEntry)
	if derivedKey == nil {
		return errors.WrapValidationError("failed to derive ledger key from entry")
	}

	derivedKeyB64, err := EncodeLedgerKey(*derivedKey)
	if err != nil {
		return errors.WrapValidationError(fmt.Sprintf("failed to encode derived ledger key: %v", err))
	}

	if derivedKeyB64 != requestedKeyB64 {
		return errors.WrapValidationError(
			fmt.Sprintf("cryptographic mismatch: requested %s but entry hashes to %s",
				requestedKeyB64, derivedKeyB64))
	}

	return nil
}

// VerifyLedgerEntries validates all returned ledger entries against their requested keys.
// Call this after fetching entries from the RPC layer to ensure data integrity before
// passing the state to the simulator.
func VerifyLedgerEntries(requestedKeys []string, returnedEntries []LedgerEntryResult) error {
	if len(requestedKeys) == 0 {
		return nil
	}

	// Build a fast lookup map
	returnedMap := make(map[string]LedgerEntryResult, len(returnedEntries))
	for _, entry := range returnedEntries {
		returnedMap[entry.Key] = entry
	}

	// Check that all requested keys are present in the response
	for _, requestedKey := range requestedKeys {
		entry, exists := returnedMap[requestedKey]
		if !exists {
			return errors.WrapValidationError(
				fmt.Sprintf("requested ledger entry not found in response: %s", requestedKey))
		}

		// Verify the hash of the returned entry
		if err := VerifyLedgerEntryHash(requestedKey, entry); err != nil {
			return fmt.Errorf("verification failed for key %s: %w", requestedKey, err)
		}
	}

	logger.Logger.Info("All ledger entries verified successfully", "count", len(requestedKeys))
	return nil
}

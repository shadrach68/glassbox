// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dotandev/glassbox/internal/errors"
	"github.com/spf13/cobra"
)

var (
	keygenOutputDir string
	keygenKeyName   string
	keygenRotate    bool
	keygenForce     bool
)

var auditKeygenCmd = &cobra.Command{
	Use:     "audit:keygen",
	GroupID: "utility",
	Short:   "Generate Ed25519 audit signing keys",
	Long: `Generate an Ed25519 key pair for audit log signing and export PEM files.

Two files are written to the output directory:
  <key-name>.private.pem   PKCS#8 PEM-encoded private key
  <key-name>.public.pem    SPKI  PEM-encoded public key

KEY ROTATION
  Use --rotate to generate a new key pair alongside existing keys.
  Keep the previous public key on disk so that audit logs signed with
  the old key can still be verified after rotation.

EXAMPLES
  # Generate default key files in the current directory
  glassbox audit:keygen

  # Generate keys with a custom name in a specific directory
  glassbox audit:keygen --key-name audit-2026 --output-dir ./keys

  # Rotate: generate new keys without removing the old pair
  glassbox audit:keygen --key-name audit-2026-new --rotate`,
	Args:    cobra.NoArgs,
	PreRunE: validateKeygenArgs,
	RunE:    runAuditKeygen,
}

func init() {
	auditKeygenCmd.Flags().StringVar(&keygenOutputDir, "output-dir", ".",
		"Directory to write generated key files")
	auditKeygenCmd.Flags().StringVar(&keygenKeyName, "key-name", "audit-signing-key",
		"Base filename for the generated key files (without extension)")
	auditKeygenCmd.Flags().BoolVar(&keygenRotate, "rotate", false,
		"Generate a new key pair alongside any existing keys for rotation")
	auditKeygenCmd.Flags().BoolVar(&keygenForce, "force", false,
		"Overwrite existing key files without prompting")

	rootCmd.AddCommand(auditKeygenCmd)
}

func validateKeygenArgs(_ *cobra.Command, _ []string) error {
	if keygenKeyName == "" {
		return errors.WrapValidationError("--key-name cannot be empty")
	}
	return nil
}

func runAuditKeygen(cmd *cobra.Command, _ []string) error {
	if err := os.MkdirAll(keygenOutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	privPath := filepath.Join(keygenOutputDir, keygenKeyName+".private.pem")
	pubPath := filepath.Join(keygenOutputDir, keygenKeyName+".public.pem")

	if !keygenForce {
		for _, p := range []string{privPath, pubPath} {
			if _, err := os.Stat(p); err == nil {
				return errors.WrapValidationError(
					fmt.Sprintf("key file already exists (use --force to overwrite): %s", p),
				)
			}
		}
	}

	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate Ed25519 key pair: %w", err)
	}

	privPEM, err := marshalPrivateKeyPEM(privKey)
	if err != nil {
		return err
	}
	pubPEM, err := marshalPublicKeyPEM(pubKey)
	if err != nil {
		return err
	}

	if err := writeKeyFile(privPath, privPEM, 0600); err != nil {
		return fmt.Errorf("failed to write private key file: %w", err)
	}
	if err := writeKeyFile(pubPath, pubPEM, 0644); err != nil {
		return fmt.Errorf("failed to write public key file: %w", err)
	}

	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "Key pair generated successfully.")
	fmt.Fprintf(out, "  Private key (PKCS#8 PEM): %s\n", privPath)
	fmt.Fprintf(out, "  Public key  (SPKI PEM):   %s\n", pubPath)
	if keygenRotate {
		fmt.Fprintln(out, "\nRotation note: keep the previous public key to verify existing signatures.")
	}
	return nil
}

func marshalPrivateKeyPEM(key ed25519.PrivateKey) ([]byte, error) {
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private key to PKCS#8: %w", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), nil
}

func marshalPublicKeyPEM(key ed25519.PublicKey) ([]byte, error) {
	der, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal public key to SPKI: %w", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}), nil
}

func writeKeyFile(path string, data []byte, mode os.FileMode) error {
	return os.WriteFile(path, data, mode)
}

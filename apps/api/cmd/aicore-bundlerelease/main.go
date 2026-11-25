// Command aicore-bundlerelease signs a clinically reviewed, non-synthetic bundle.
// Never commit the private key or put it inside the bundle directory.
package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/bundle"
)

func main() {
	dir := flag.String("dir", "knowledge", "non-synthetic bundle directory")
	core := flag.String("core-version", "1.0.0", "current AI core version")
	keyID := flag.String("key-id", "", "trusted release-signing key identifier")
	keyFile := flag.String("private-key-file", "", "private Ed25519 signing key (base64, outside repository)")
	flag.Parse()
	if *keyFile == "" || *keyID == "" {
		fmt.Fprintln(os.Stderr, "key-id and private-key-file are required")
		os.Exit(2)
	}
	bundleRoot, err := filepath.Abs(*dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "invalid bundle path")
		os.Exit(2)
	}
	keyPath, err := filepath.Abs(*keyFile)
	if err != nil || strings.HasPrefix(keyPath, bundleRoot+string(os.PathSeparator)) {
		fmt.Fprintln(os.Stderr, "private key cannot live inside bundle directory")
		os.Exit(2)
	}
	raw, err := os.ReadFile(keyPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "unable to read private key")
		os.Exit(2)
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil || len(decoded) != ed25519.PrivateKeySize {
		fmt.Fprintln(os.Stderr, "invalid Ed25519 private key")
		os.Exit(2)
	}
	if err := bundle.SignRelease(context.Background(), *dir, *core, *keyID, ed25519.PrivateKey(decoded)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("release manifest checksummed and Ed25519-signed")
}

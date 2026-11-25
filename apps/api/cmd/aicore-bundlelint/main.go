// Command aicore-bundlelint validates a Knowledge Bundle without running models.
package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/bundle"
)

func main() {
	dir := flag.String("dir", "knowledge", "bundle root, relative to repository root by default")
	core := flag.String("core-version", "1.0.0", "current AI core semantic version")
	release := flag.Bool("release", false, "enforce clinical sign-off, current safety resources, verified hashes and trusted signature")
	keyArg := flag.String("trusted-key", "", "release verify key: key-id:base64-ed25519-public-key")
	flag.Parse()
	keys := map[string]ed25519.PublicKey{}
	if *keyArg != "" {
		parts := strings.SplitN(*keyArg, ":", 2)
		if len(parts) != 2 {
			fmt.Fprintln(os.Stderr, "invalid trusted-key format")
			os.Exit(2)
		}
		raw, err := base64.StdEncoding.DecodeString(parts[1])
		if err != nil || len(raw) != ed25519.PublicKeySize {
			fmt.Fprintln(os.Stderr, "invalid public key")
			os.Exit(2)
		}
		keys[parts[0]] = ed25519.PublicKey(raw)
	}
	snap, report, err := bundle.Read(context.Background(), *dir, bundle.Options{CoreVersion: *core, Release: *release, TrustedKeys: keys})
	for _, issue := range report.Warnings {
		fmt.Fprintf(os.Stderr, "warning %s [%s]: %s\n", issue.Path, issue.Code, issue.Message)
	}
	for _, issue := range report.Errors {
		fmt.Fprintf(os.Stderr, "error %s [%s]: %s\n", issue.Path, issue.Code, issue.Message)
	}
	if err != nil {
		os.Exit(1)
	}
	info := map[string]string{"version": snap.Version(), "sha256": snap.Fingerprint(), "status": "validated"}
	encoded, _ := json.Marshal(info)
	fmt.Println(string(encoded))
}

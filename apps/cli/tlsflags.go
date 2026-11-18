package main

import (
	"crypto/tls"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/fuvito/api-workbench/packages/core/executor"
)

// tlsFlags holds the mutual-TLS files given on the command line. They are paths only: the key and
// certificate are read when a run starts, never stored or printed.
var tlsFlags executor.TLSFiles

func addTLSFlags(root *cobra.Command) {
	f := root.PersistentFlags()
	f.StringVar(&tlsFlags.ClientCertFile, "client-cert", "", "PEM client certificate for mutual TLS (or set CLRSIG_CLIENT_CERT)")
	f.StringVar(&tlsFlags.ClientKeyFile, "client-key", "", "PEM private key for --client-cert, not password-protected (or set CLRSIG_CLIENT_KEY)")
	f.StringVar(&tlsFlags.CAFile, "ca-cert", "", "PEM file of extra CA certificates to trust (or set CLRSIG_CA_CERT)")
	root.PersistentPreRunE = func(*cobra.Command, []string) error {
		files := effectiveTLS()
		if files.IsZero() {
			return nil
		}
		// load once up front so a wrong path stops the command before any request is sent
		return files.Apply(&tls.Config{}) //nolint:gosec
	}
}

// effectiveTLS is the command-line value, falling back to the CLRSIG_* environment variable.
func effectiveTLS() executor.TLSFiles {
	pick := func(flag, env string) string {
		if flag != "" {
			return flag
		}
		return strings.TrimSpace(os.Getenv(env))
	}
	return executor.TLSFiles{
		ClientCertFile: pick(tlsFlags.ClientCertFile, "CLRSIG_CLIENT_CERT"),
		ClientKeyFile:  pick(tlsFlags.ClientKeyFile, "CLRSIG_CLIENT_KEY"),
		CAFile:         pick(tlsFlags.CAFile, "CLRSIG_CA_CERT"),
	}
}

// newExecutor is the one place the CLI builds an HTTP executor.
func newExecutor() *executor.Executor {
	return executor.New(executor.Options{TLS: effectiveTLS()})
}

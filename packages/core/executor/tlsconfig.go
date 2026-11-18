package executor

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"

	"github.com/fuvito/api-workbench/packages/core/variables"
)

// TLSFiles names the certificate files for mutual TLS. Only paths are ever stored or passed
// around: the key and certificate contents are read when a run starts. A path may contain ${NAME}
// references, which are expanded from the process environment at that moment.
type TLSFiles struct {
	ClientCertFile string // PEM client certificate (or chain); needs ClientKeyFile
	ClientKeyFile  string // PEM private key, not password-protected
	CAFile         string // PEM bundle of extra CA certificates to trust, added to the system roots
}

// IsZero reports whether no certificate file is configured.
func (t TLSFiles) IsZero() bool {
	return t.ClientCertFile == "" && t.ClientKeyFile == "" && t.CAFile == ""
}

// Apply loads the files into cfg. It returns a clear error naming the setting and the file, so a
// wrong path is reported before any request is sent.
func (t TLSFiles) Apply(cfg *tls.Config) error {
	cert, key := expandPath(t.ClientCertFile), expandPath(t.ClientKeyFile)
	if (cert == "") != (key == "") {
		return fmt.Errorf("client certificate: set both the certificate file and the key file")
	}
	if cert != "" {
		pair, err := tls.LoadX509KeyPair(cert, key)
		if err != nil {
			return fmt.Errorf("client certificate %q with key %q: %w", cert, key, err)
		}
		cfg.Certificates = []tls.Certificate{pair}
	}
	if ca := expandPath(t.CAFile); ca != "" {
		pem, err := os.ReadFile(ca)
		if err != nil {
			return fmt.Errorf("CA certificate file: %w", err)
		}
		pool, err := x509.SystemCertPool()
		if err != nil || pool == nil {
			pool = x509.NewCertPool()
		}
		if !pool.AppendCertsFromPEM(pem) {
			return fmt.Errorf("CA certificate file %q contains no PEM certificate", ca)
		}
		cfg.RootCAs = pool
	}
	return nil
}

func expandPath(p string) string { return strings.TrimSpace(variables.ExpandEnv(p)) }

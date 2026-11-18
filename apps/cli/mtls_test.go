package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeBlock(t *testing.T, path, typ string, der []byte) {
	t.Helper()
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: typ, Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}
}

// mtlsFixture starts a TLS server that requires a client certificate and returns the files a
// client needs: certificate, key and the server's own certificate as the CA to trust.
func mtlsFixture(t *testing.T) (url, cert, key, ca string) {
	t.Helper()
	dir := t.TempDir()
	caKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	caTmpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "ca"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign}
	caDER, _ := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &caKey.PublicKey, caKey)
	caCert, _ := x509.ParseCertificate(caDER)
	pool := x509.NewCertPool()
	pool.AddCert(caCert)

	clKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	clTmpl := &x509.Certificate{SerialNumber: big.NewInt(2), Subject: pkix.Name{CommonName: "client"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}}
	clDER, _ := x509.CreateCertificate(rand.Reader, clTmpl, caCert, &clKey.PublicKey, caKey)
	keyDER, _ := x509.MarshalECPrivateKey(clKey)

	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"client":%q}`, r.TLS.PeerCertificates[0].Subject.CommonName)
	}))
	srv.TLS = &tls.Config{ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs: pool}
	srv.StartTLS()
	t.Cleanup(srv.Close)

	cert, key, ca = filepath.Join(dir, "client.pem"), filepath.Join(dir, "client.key"), filepath.Join(dir, "ca.pem")
	writeBlock(t, cert, "CERTIFICATE", clDER)
	writeBlock(t, key, "EC PRIVATE KEY", keyDER)
	writeBlock(t, ca, "CERTIFICATE", srv.Certificate().Raw)
	return srv.URL, cert, key, ca
}

const mtlsScenario = `schemaVersion: "1"
title: Mutual TLS
steps:
  - label: who am I
    request: {method: GET, url: "%s/me"}
    assertions:
      - {type: jsonpath, path: $.client, operator: eq, expected: client}
`

func TestRun_ClientCertificateFlags(t *testing.T) {
	url, cert, key, ca := mtlsFixture(t)
	sc := writeTempFile(t, "m.yaml", fmt.Sprintf(mtlsScenario, url))

	if out, err := exec(t, "run", "--client-cert", cert, "--client-key", key, "--ca-cert", ca, sc); err != nil {
		t.Fatalf("with the certificate the server accepts the client: %v\n%s", err, out)
	}
	if out, err := exec(t, "run", "--ca-cert", ca, sc); exitCode(err) != 1 {
		t.Errorf("without a client certificate the request fails (exit 1): %v\n%s", err, out)
	}
}

func TestRun_ClientCertificateFromEnvironmentVariables(t *testing.T) {
	url, cert, key, ca := mtlsFixture(t)
	sc := writeTempFile(t, "m.yaml", fmt.Sprintf(mtlsScenario, url))
	t.Setenv("CLRSIG_CLIENT_CERT", cert)
	t.Setenv("CLRSIG_CLIENT_KEY", key)
	t.Setenv("CLRSIG_CA_CERT", ca)
	if out, err := exec(t, "run", sc); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
}

func TestRun_BadCertificatePathStopsBeforeAnyRequest(t *testing.T) {
	url, cert, _, _ := mtlsFixture(t)
	sc := writeTempFile(t, "m.yaml", fmt.Sprintf(mtlsScenario, url))
	_, err := exec(t, "run", "--client-cert", cert, sc)
	if err == nil || !strings.Contains(err.Error(), "set both the certificate file and the key file") || exitCode(err) == 1 {
		t.Errorf("a half-configured certificate is a configuration error (exit 2): %v", err)
	}
	_, err = exec(t, "run", "--client-cert", cert, "--client-key", filepath.Join(t.TempDir(), "missing.key"), sc)
	if err == nil || !strings.Contains(err.Error(), "missing.key") {
		t.Errorf("a missing key file is named in the error: %v", err)
	}
}

package executor_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/executor"
)

type pki struct {
	dir                         string
	caPool                      *x509.CertPool
	clientCert, clientKey, caPE string
	serverPEM                   string
}

func writePEM(t *testing.T, path, typ string, der []byte) {
	t.Helper()
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: typ, Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}
}

// newPKI makes a CA and a client certificate it signed, written as PEM files.
func newPKI(t *testing.T) *pki {
	t.Helper()
	dir := t.TempDir()
	caKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	caTmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "test ca"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign,
	}
	caDER, _ := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &caKey.PublicKey, caKey)
	caCert, _ := x509.ParseCertificate(caDER)

	clKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	clTmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2), Subject: pkix.Name{CommonName: "client"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	clDER, _ := x509.CreateCertificate(rand.Reader, clTmpl, caCert, &clKey.PublicKey, caKey)
	keyDER, _ := x509.MarshalECPrivateKey(clKey)

	p := &pki{dir: dir, caPool: x509.NewCertPool(), clientCert: filepath.Join(dir, "client.pem"), clientKey: filepath.Join(dir, "client.key")}
	p.caPool.AddCert(caCert)
	writePEM(t, p.clientCert, "CERTIFICATE", clDER)
	writePEM(t, p.clientKey, "EC PRIVATE KEY", keyDER)
	return p
}

// mtlsServer requires a client certificate signed by p's CA and writes the CA file the client
// needs to trust the server's own self-signed certificate.
func mtlsServer(t *testing.T, p *pki) *httptest.Server {
	t.Helper()
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(r.TLS.PeerCertificates[0].Subject.CommonName))
	}))
	srv.TLS = &tls.Config{ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs: p.caPool}
	srv.StartTLS()
	t.Cleanup(srv.Close)
	p.serverPEM = filepath.Join(p.dir, "server.pem")
	writePEM(t, p.serverPEM, "CERTIFICATE", srv.Certificate().Raw)
	return srv
}

func get(t *testing.T, opts executor.Options, url string) (*domain.Response, error) {
	t.Helper()
	return executor.New(opts).Execute(context.Background(), &domain.Request{Method: domain.MethodGET, URL: url})
}

func TestClientCertificateIsPresented(t *testing.T) {
	p := newPKI(t)
	srv := mtlsServer(t, p)

	resp, err := get(t, executor.Options{TLS: executor.TLSFiles{ClientCertFile: p.clientCert, ClientKeyFile: p.clientKey, CAFile: p.serverPEM}}, srv.URL)
	if err != nil || resp.StatusCode != 200 || resp.BodyString() != "client" {
		t.Fatalf("the server should see the client certificate: %v %+v", err, resp)
	}

	if _, err := get(t, executor.Options{TLS: executor.TLSFiles{CAFile: p.serverPEM}}, srv.URL); err == nil {
		t.Error("without a client certificate the server must refuse the connection")
	}
}

func TestCertificatePathsMayUseEnvironmentReferences(t *testing.T) {
	p := newPKI(t)
	srv := mtlsServer(t, p)
	t.Setenv("CERT_DIR", p.dir)
	files := executor.TLSFiles{ClientCertFile: "${CERT_DIR}/client.pem", ClientKeyFile: "${CERT_DIR}/client.key", CAFile: "${CERT_DIR}/server.pem"}
	if resp, err := get(t, executor.Options{TLS: files}, srv.URL); err != nil || resp.StatusCode != 200 {
		t.Fatalf("%v %+v", err, resp)
	}
}

func TestBadCertificateSettingsAreReportedClearly(t *testing.T) {
	p := newPKI(t)
	bad := filepath.Join(p.dir, "bad.pem")
	os.WriteFile(bad, []byte("not pem"), 0o600)
	for name, tc := range map[string]struct {
		files executor.TLSFiles
		want  string
	}{
		"cert without key": {executor.TLSFiles{ClientCertFile: p.clientCert}, "set both the certificate file and the key file"},
		"missing file":     {executor.TLSFiles{ClientCertFile: filepath.Join(p.dir, "nope.pem"), ClientKeyFile: p.clientKey}, "nope.pem"},
		"not a pem":        {executor.TLSFiles{CAFile: bad}, "no PEM certificate"},
		"missing ca":       {executor.TLSFiles{CAFile: filepath.Join(p.dir, "gone.pem")}, "CA certificate file"},
	} {
		_, err := get(t, executor.Options{TLS: tc.files}, "https://127.0.0.1:1")
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: error %v, want it to contain %q", name, err, tc.want)
		}
	}
}

func TestNoCertificateSettingsChangeNothing(t *testing.T) {
	if !(executor.TLSFiles{}).IsZero() {
		t.Error("empty files are zero")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) }))
	defer srv.Close()
	if resp, err := get(t, executor.Options{}, srv.URL); err != nil || resp.BodyString() != "ok" {
		t.Fatal(err)
	}
}

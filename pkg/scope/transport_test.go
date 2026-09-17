/*
Copyright 2026 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package scope

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"sync"
	"testing"
	"time"
)

// generateTestCACertPEM returns a self-signed CA certificate in PEM form,
// suitable for exercising the x509.CertPool.AppendCertsFromPEM success path.
func generateTestCACertPEM(t *testing.T) []byte {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test-ca"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
		IsCA:         true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

func boolPtr(b bool) *bool { return &b }

// None of these tests use t.Parallel(): they share the package-level,
// size-bounded transportCache, and running them concurrently would let one
// test's insertions evict entries another test is mid-assertion on.

func TestGetOrCreateTransport_SameConfigReturnsSameInstance(t *testing.T) {
	caCert := generateTestCACertPEM(t)

	t1, err := getOrCreateTransport(caCert, boolPtr(true))
	if err != nil {
		t.Fatalf("getOrCreateTransport: %v", err)
	}
	t2, err := getOrCreateTransport(caCert, boolPtr(true))
	if err != nil {
		t.Fatalf("getOrCreateTransport: %v", err)
	}

	if t1 != t2 {
		t.Fatalf("expected the same *http.Transport instance for identical TLS config, got distinct instances")
	}
}

func TestGetOrCreateTransport_DifferentCACertReturnsDifferentInstance(t *testing.T) {
	cert1 := generateTestCACertPEM(t)
	cert2 := generateTestCACertPEM(t)

	t1, err := getOrCreateTransport(cert1, boolPtr(true))
	if err != nil {
		t.Fatalf("getOrCreateTransport: %v", err)
	}
	t2, err := getOrCreateTransport(cert2, boolPtr(true))
	if err != nil {
		t.Fatalf("getOrCreateTransport: %v", err)
	}

	if t1 == t2 {
		t.Fatalf("expected distinct *http.Transport instances for different CA certs, got the same instance")
	}
}

func TestGetOrCreateTransport_DifferentVerifyReturnsDifferentInstance(t *testing.T) {
	caCert := generateTestCACertPEM(t)

	insecure, err := getOrCreateTransport(caCert, boolPtr(false))
	if err != nil {
		t.Fatalf("getOrCreateTransport: %v", err)
	}
	secure, err := getOrCreateTransport(caCert, boolPtr(true))
	if err != nil {
		t.Fatalf("getOrCreateTransport: %v", err)
	}
	unset, err := getOrCreateTransport(caCert, nil)
	if err != nil {
		t.Fatalf("getOrCreateTransport: %v", err)
	}

	if insecure == secure || insecure == unset || secure == unset {
		t.Fatalf("expected distinct *http.Transport instances for verify=false/true/nil, got overlapping instances")
	}
}

func TestGetOrCreateTransport_TLSConfigReflectsVerifyAndCACert(t *testing.T) {
	validCert := generateTestCACertPEM(t)

	transport, err := getOrCreateTransport(validCert, boolPtr(false))
	if err != nil {
		t.Fatalf("getOrCreateTransport: %v", err)
	}
	if !transport.TLSClientConfig.InsecureSkipVerify {
		t.Fatalf("expected InsecureSkipVerify=true when verify=false")
	}
	if transport.TLSClientConfig.RootCAs == nil {
		t.Fatalf("expected RootCAs to be populated for a valid CA cert")
	}

	verifyingTransport, err := getOrCreateTransport(validCert, boolPtr(true))
	if err != nil {
		t.Fatalf("getOrCreateTransport: %v", err)
	}
	if verifyingTransport.TLSClientConfig.InsecureSkipVerify {
		t.Fatalf("expected InsecureSkipVerify=false when verify=true")
	}
}

func TestGetOrCreateTransport_InvalidCACertFallsBackToNilRootCAs(t *testing.T) {
	// Not valid PEM/DER - AppendCertsFromPEM will fail to parse any certs.
	invalidCert := []byte("-----BEGIN CERTIFICATE-----\nnot-a-real-cert\n-----END CERTIFICATE-----\n")

	transport, err := getOrCreateTransport(invalidCert, boolPtr(true))
	if err != nil {
		t.Fatalf("getOrCreateTransport: %v", err)
	}
	if transport.TLSClientConfig.RootCAs != nil {
		t.Fatalf("expected nil RootCAs when no certs could be parsed from caCert, falling back to host root CAs")
	}
}

func TestGetOrCreateTransport_HasFleetSuitablePoolingDefaults(t *testing.T) {
	transport, err := getOrCreateTransport(generateTestCACertPEM(t), boolPtr(true))
	if err != nil {
		t.Fatalf("getOrCreateTransport: %v", err)
	}
	if transport.IdleConnTimeout <= 0 {
		t.Fatalf("expected a non-zero IdleConnTimeout so idle connections in the shared transport self-expire, got %v", transport.IdleConnTimeout)
	}
	// Go's unset default is 2, which is too low once this Transport is
	// shared by every credential in the fleet against the same host.
	if transport.MaxIdleConnsPerHost <= 2 {
		t.Fatalf("expected MaxIdleConnsPerHost to be raised above the net/http default of 2 for a fleet-shared transport, got %d", transport.MaxIdleConnsPerHost)
	}
}

// TestGetOrCreateTransport_ConcurrentAccessReturnsSingleInstance guards against
// races in the shared transportCache: concurrent first-time callers for the
// same TLS config must still converge on exactly one *http.Transport, since
// that transport's connection pool is what makes eviction of the (much
// higher-cardinality) per-credential scope cache safe.
func TestGetOrCreateTransport_ConcurrentAccessReturnsSingleInstance(t *testing.T) {
	caCert := generateTestCACertPEM(t)

	const goroutines = 50
	type result struct {
		transport *http.Transport
		err       error
	}
	out := make([]result, goroutines)

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := range goroutines {
		go func(i int) {
			defer wg.Done()
			tr, err := getOrCreateTransport(caCert, boolPtr(true))
			out[i] = result{transport: tr, err: err}
		}(i)
	}
	wg.Wait()

	first := out[0]
	if first.err != nil {
		t.Fatalf("getOrCreateTransport: %v", first.err)
	}
	for i, r := range out {
		if r.err != nil {
			t.Fatalf("goroutine %d: getOrCreateTransport: %v", i, r.err)
		}
		if r.transport != first.transport {
			t.Fatalf("goroutine %d returned a different *http.Transport instance than goroutine 0", i)
		}
	}
}

// TestGetOrCreateTransport_EvictsLeastRecentlyUsedWhenFull exercises the
// cache's bound: pushing it past maxCachedTransports must evict the least
// recently used entry (closing its idle connections), not an arbitrary one,
// and a recently-touched entry must survive.
func TestGetOrCreateTransport_EvictsLeastRecentlyUsedWhenFull(t *testing.T) {
	certs := make([][]byte, maxCachedTransports)
	transports := make([]*http.Transport, maxCachedTransports)
	for i := range certs {
		certs[i] = generateTestCACertPEM(t)
		tr, err := getOrCreateTransport(certs[i], boolPtr(true))
		if err != nil {
			t.Fatalf("getOrCreateTransport(%d): %v", i, err)
		}
		transports[i] = tr
	}

	// Touch index 0 so it becomes most-recently-used; index 1 becomes the
	// least-recently-used and should be the one evicted below.
	if tr, err := getOrCreateTransport(certs[0], boolPtr(true)); err != nil || tr != transports[0] {
		t.Fatalf("expected a cache hit for certs[0] before overflow, got transport=%v err=%v", tr, err)
	}

	// One more distinct config pushes the cache past its bound.
	overflowCert := generateTestCACertPEM(t)
	if _, err := getOrCreateTransport(overflowCert, boolPtr(true)); err != nil {
		t.Fatalf("getOrCreateTransport(overflow): %v", err)
	}

	if tr, err := getOrCreateTransport(certs[0], boolPtr(true)); err != nil || tr != transports[0] {
		t.Fatalf("expected certs[0] (recently used) to survive eviction, got transport=%v err=%v (want %v)", tr, err, transports[0])
	}
	if tr, err := getOrCreateTransport(certs[1], boolPtr(true)); err != nil || tr == transports[1] {
		t.Fatalf("expected certs[1] (least recently used) to have been evicted and replaced, got the original instance back (err=%v)", err)
	}
}

// TestSetMaxCachedTransports checks that the bound can be overridden (as
// NewFactory does from --transport-cache-max-size), and that non-positive
// values are ignored rather than disabling the cache - unlike the scope
// cache, this one must always stay on for eviction to remain safe.
func TestSetMaxCachedTransports(t *testing.T) {
	original := maxCachedTransports
	t.Cleanup(func() { setMaxCachedTransports(original) })

	setMaxCachedTransports(3)
	if maxCachedTransports != 3 {
		t.Fatalf("expected maxCachedTransports to be updated to 3, got %d", maxCachedTransports)
	}

	setMaxCachedTransports(0)
	if maxCachedTransports != 3 {
		t.Fatalf("expected setMaxCachedTransports(0) to be ignored, got %d", maxCachedTransports)
	}

	setMaxCachedTransports(-5)
	if maxCachedTransports != 3 {
		t.Fatalf("expected setMaxCachedTransports(-5) to be ignored, got %d", maxCachedTransports)
	}
}

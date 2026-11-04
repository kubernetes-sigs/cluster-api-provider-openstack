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
	"container/list"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// maxCachedTransports bounds how many distinct TLS trust configurations
// (CA bundle + verify mode) are kept warm at once. Overridable via
// setMaxCachedTransports / --transport-cache-max-size.
var maxCachedTransports = 10

// setMaxCachedTransports cannot disable the cache (n <= 0 is ignored):
// evicting an entry here is what closes its connections, so it must stay on
// for scope-cache eviction elsewhere to be safe.
func setMaxCachedTransports(n int) {
	if n < 1 {
		return
	}
	transportCacheMu.Lock()
	defer transportCacheMu.Unlock()
	maxCachedTransports = n
}

var transportCacheMu sync.Mutex

// transportCacheOrder/transportCacheIndex form a small LRU of *http.Transport
// keyed by TLS trust configuration (CA bundle + verify mode) rather than by
// credential. net/http never closes a Transport's idle connections on its
// own, so caching one per credential (high-cardinality, evicted constantly
// by the scope cache) leaked a connection pool on every eviction. TLS
// config is low-cardinality and stable across a fleet on the same cloud(s),
// so it's safe to cache here instead, explicitly calling
// CloseIdleConnections on whatever gets evicted on overflow.
var (
	transportCacheOrder list.List
	transportCacheIndex = map[string]*list.Element{}
)

type transportCacheEntry struct {
	key       string
	transport *http.Transport
}

func getOrCreateTransport(caCert []byte, verify *bool) (*http.Transport, error) {
	key, err := transportCacheKey(caCert, verify)
	if err != nil {
		return nil, fmt.Errorf("compute transport cache key: %w", err)
	}

	transportCacheMu.Lock()
	defer transportCacheMu.Unlock()

	if element, ok := transportCacheIndex[key]; ok {
		transportCacheOrder.MoveToFront(element)
		return element.Value.(*transportCacheEntry).transport, nil
	}

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	if verify != nil {
		tlsConfig.InsecureSkipVerify = !*verify
	}
	if caCert != nil {
		tlsConfig.RootCAs = x509.NewCertPool()
		if !tlsConfig.RootCAs.AppendCertsFromPEM(caCert) {
			// If no certificates were successfully parsed, set RootCAs to nil to use the host's root CA
			tlsConfig.RootCAs = nil
		}
	}

	t := &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		TLSClientConfig:     tlsConfig,
		IdleConnTimeout:     90 * time.Second,
		MaxIdleConnsPerHost: 25,
	}

	if transportCacheOrder.Len() >= maxCachedTransports {
		oldest := transportCacheOrder.Back()
		oldestEntry := oldest.Value.(*transportCacheEntry)
		transportCacheOrder.Remove(oldest)
		delete(transportCacheIndex, oldestEntry.key)
		oldestEntry.transport.CloseIdleConnections()
	}

	transportCacheIndex[key] = transportCacheOrder.PushFront(&transportCacheEntry{key: key, transport: t})
	return t, nil
}

type transportCacheKeyInput struct {
	CACert []byte
	Verify *bool
}

func transportCacheKey(caCert []byte, verify *bool) (string, error) {
	hash, err := computeSpewHash(transportCacheKeyInput{CACert: caCert, Verify: verify})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d", hash), nil
}

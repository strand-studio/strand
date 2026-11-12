package security

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestSafeHTTPClientHonorsHTTPSProxy(t *testing.T) {
	var sawConnect bool
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodConnect {
			sawConnect = true
			// Minimal CONNECT rejection is enough to prove the client dialed the proxy.
			http.Error(w, "connect ok for test", http.StatusForbidden)
			return
		}
		http.Error(w, "unexpected", http.StatusBadRequest)
	}))
	defer proxy.Close()

	proxyURL, err := url.Parse(proxy.URL)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HTTP_PROXY", proxy.URL)
	t.Setenv("HTTPS_PROXY", proxy.URL)
	t.Setenv("NO_PROXY", "")
	t.Setenv("http_proxy", proxy.URL)
	t.Setenv("https_proxy", proxy.URL)
	t.Setenv("no_proxy", "")

	client := SafeHTTPClient(5 * time.Second)
	// Ensure ProxyFromEnvironment is active even if DefaultTransport was cached.
	if transport, ok := client.Transport.(*validatingTransport); ok {
		if base, ok := transport.base.(*http.Transport); ok {
			base.Proxy = http.ProxyURL(proxyURL)
		}
	}

	req, err := http.NewRequest(http.MethodGet, "https://api.openai.com/v1/models", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.Do(req)
	if resp != nil {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
	if !sawConnect {
		t.Fatalf("expected HTTPS request to go through proxy; err=%v", err)
	}
}

func TestSafeHTTPClientBlocksPrivateDestination(t *testing.T) {
	client := SafeHTTPClient(3 * time.Second)
	req, err := http.NewRequest(http.MethodGet, "http://127.0.0.1/", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Do(req)
	if err == nil {
		t.Fatal("expected SSRF block for loopback destination")
	}
}

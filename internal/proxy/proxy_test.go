package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestProxyForwardsRequestAndResponse(t *testing.T) {
	var gotMethod, gotPath, gotQuery, gotHeader, gotBody, gotForwardedHost, gotForwardedFor string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		gotHeader = r.Header.Get("X-Client")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		gotBody = string(body)
		gotForwardedHost = r.Header.Get("X-Forwarded-Host")
		gotForwardedFor = r.Header.Get("X-Forwarded-For")
		w.Header().Set("X-Upstream", "ok")
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, "from-upstream")
	}))
	defer upstream.Close()

	target, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatal(err)
	}
	proxyServer := httptest.NewServer(New(target, AllowAll{}))
	defer proxyServer.Close()

	request, err := http.NewRequest(http.MethodPost, proxyServer.URL+"/orders?id=7", strings.NewReader("sku=a"))
	if err != nil {
		t.Fatal(err)
	}
	request.Host = "shop.example"
	request.Header.Set("X-Client", "storefront")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}

	if gotMethod != http.MethodPost || gotPath != "/orders" || gotQuery != "id=7" || gotHeader != "storefront" || gotBody != "sku=a" {
		t.Fatalf("upstream saw method=%s path=%s query=%s header=%s body=%s", gotMethod, gotPath, gotQuery, gotHeader, gotBody)
	}
	if gotForwardedHost != "shop.example" {
		t.Fatalf("X-Forwarded-Host = %q", gotForwardedHost)
	}
	if gotForwardedFor == "" {
		t.Fatal("X-Forwarded-For was empty")
	}
	if response.StatusCode != http.StatusCreated || string(responseBody) != "from-upstream" || response.Header.Get("X-Upstream") != "ok" {
		t.Fatalf("client status=%d body=%q header=%q", response.StatusCode, responseBody, response.Header.Get("X-Upstream"))
	}
}

func TestProxyStopsBeforeUpstreamWhenInspectionDenies(t *testing.T) {
	called := false
	upstream := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))
	defer upstream.Close()
	target, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatal(err)
	}
	proxyServer := httptest.NewServer(New(target, denyPath{path: "/admin"}))
	defer proxyServer.Close()

	response, err := http.Get(proxyServer.URL + "/admin")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d", response.StatusCode)
	}
	if called {
		t.Fatal("upstream was called after a denied inspection")
	}
}

func TestProxyReturnsBadGatewayWhenUpstreamIsDown(t *testing.T) {
	upstream := httptest.NewServer(http.NotFoundHandler())
	target, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatal(err)
	}
	upstream.Close()

	proxyServer := httptest.NewServer(New(target, AllowAll{}))
	defer proxyServer.Close()

	response, err := http.Get(proxyServer.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusBadGateway {
		t.Fatalf("status = %d", response.StatusCode)
	}
}

func TestIngressRoutesByHost(t *testing.T) {
	shop := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "shop:"+r.URL.Path)
	}))
	defer shop.Close()
	pay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "pay:"+r.URL.Path)
	}))
	defer pay.Close()
	shopURL, err := url.Parse(shop.URL)
	if err != nil {
		t.Fatal(err)
	}
	payURL, err := url.Parse(pay.URL)
	if err != nil {
		t.Fatal(err)
	}
	proxyServer := httptest.NewServer(NewIngress(map[string]*url.URL{
		"shop.example": shopURL,
		"pay.example":  payURL,
	}, AllowAll{}))
	defer proxyServer.Close()

	if got := getHost(t, proxyServer.URL+"/cart", "shop.example"); got != "shop:/cart" {
		t.Fatalf("shop body = %q", got)
	}
	if got := getHost(t, proxyServer.URL+"/charge", "pay.example"); got != "pay:/charge" {
		t.Fatalf("pay body = %q", got)
	}
	response, err := getHostResponse(t, proxyServer.URL+"/missing", "unknown.example")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d", response.StatusCode)
	}
}

func TestBridgeForwardsToOriginalDestination(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, r.URL.Path)
	}))
	defer upstream.Close()
	target, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "http://waf.example/orders", nil)
	request = request.WithContext(WithDestination(request.Context(), target))
	recorder := httptest.NewRecorder()
	NewBridge(AllowAll{}).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || recorder.Body.String() != "/orders" {
		t.Fatalf("status=%d body=%q", recorder.Code, recorder.Body.String())
	}

	missing := httptest.NewRecorder()
	NewBridge(AllowAll{}).ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "http://waf.example/orders", nil))
	if missing.Code != http.StatusBadGateway {
		t.Fatalf("status = %d", missing.Code)
	}
}

func getHost(t *testing.T, rawURL, host string) string {
	t.Helper()
	response, err := getHostResponse(t, rawURL, host)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d body = %s", response.StatusCode, body)
	}
	return string(body)
}

func getHostResponse(t *testing.T, rawURL, host string) (*http.Response, error) {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Host = host
	return http.DefaultClient.Do(request)
}

type denyPath struct {
	path string
}

func (d denyPath) Inspect(r *http.Request) Result {
	return Result{Allow: r.URL.Path != d.path}
}

package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

// Inspector sees each request before it is forwarded.
// Inspect must not read or close Request.Body. Body inspection belongs to a later task.
type Inspector interface {
	Inspect(*http.Request) Result
}

// Result is the outcome of the inline check.
type Result struct {
	Allow bool
}

// AllowAll passes every request through. It is the WAF-1 inspection point.
type AllowAll struct{}

func (AllowAll) Inspect(*http.Request) Result { return Result{Allow: true} }

// New returns a reverse proxy that inspects a request and then forwards it to upstream.
func New(upstream *url.URL, inspector Inspector) http.Handler {
	if inspector == nil {
		inspector = AllowAll{}
	}
	reverse := &httputil.ReverseProxy{
		Rewrite: func(request *httputil.ProxyRequest) {
			request.SetURL(upstream)
			request.SetXForwarded()
		},
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, _ error) {
			http.Error(w, "업스트림에 연결하지 못했습니다", http.StatusBadGateway)
		},
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !inspector.Inspect(r).Allow {
			http.Error(w, "요청이 차단되었습니다", http.StatusForbidden)
			return
		}
		reverse.ServeHTTP(w, r)
	})
}

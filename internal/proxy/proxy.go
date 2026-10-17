package proxy

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
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

var errUnknownHost = errors.New("unknown host")
var errNoDestination = errors.New("missing original destination")

type destinationKey struct{}
type targetKey struct{}

// WithDestination records the original destination of a transparently redirected connection.
func WithDestination(ctx context.Context, upstream *url.URL) context.Context {
	return context.WithValue(ctx, destinationKey{}, upstream)
}

// New returns a reverse proxy that inspects a request and then forwards it to upstream.
func New(upstream *url.URL, inspector Inspector) http.Handler {
	return serve(func(*http.Request) (*url.URL, error) {
		return upstream, nil
	}, inspector)
}

// NewIngress forwards a request to the upstream registered for its Host header.
func NewIngress(routes map[string]*url.URL, inspector Inspector) http.Handler {
	copied := make(map[string]*url.URL, len(routes))
	for host, upstream := range routes {
		copied[canonicalHost(host)] = upstream
	}
	return serve(func(r *http.Request) (*url.URL, error) {
		upstream, ok := copied[canonicalHost(r.Host)]
		if !ok || upstream == nil {
			return nil, errUnknownHost
		}
		return upstream, nil
	}, inspector)
}

// NewBridge forwards a request to the original destination stored on the request context.
func NewBridge(inspector Inspector) http.Handler {
	return serve(func(r *http.Request) (*url.URL, error) {
		upstream, _ := r.Context().Value(destinationKey{}).(*url.URL)
		if upstream == nil {
			return nil, errNoDestination
		}
		return upstream, nil
	}, inspector)
}

func serve(resolve func(*http.Request) (*url.URL, error), inspector Inspector) http.Handler {
	if inspector == nil {
		inspector = AllowAll{}
	}
	reverse := &httputil.ReverseProxy{
		Rewrite: func(request *httputil.ProxyRequest) {
			target, _ := request.In.Context().Value(targetKey{}).(*url.URL)
			request.SetURL(target)
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
		target, err := resolve(r)
		if errors.Is(err, errUnknownHost) {
			http.Error(w, "호스트에 대한 경로가 없습니다", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "원래 목적지를 찾지 못했습니다", http.StatusBadGateway)
			return
		}
		r = r.WithContext(context.WithValue(r.Context(), targetKey{}, target))
		reverse.ServeHTTP(w, r)
	})
}

func canonicalHost(host string) string {
	host = strings.TrimSpace(host)
	if parsed, _, err := net.SplitHostPort(host); err == nil {
		host = parsed
	}
	return strings.ToLower(host)
}

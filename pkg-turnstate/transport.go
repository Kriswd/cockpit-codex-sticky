package turnstate

import (
	"net/http"
	"strings"

	cliproxyauth "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/auth"
)

// Transport wraps a base RoundTripper to capture/inject X-Codex-Turn-State
// without replacing TLS fingerprinting stacks (wrap outermost only).
// Also emits [effort-detect] logs for requested reasoning effort and
// response usage.reasoning_tokens / 516 truncation fingerprint.
type Transport struct {
	Base  http.RoundTripper
	Auth  *cliproxyauth.Auth
	Cache *Cache
}

func Wrap(base http.RoundTripper, auth *cliproxyauth.Auth) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	if auth == nil {
		return base
	}
	return &Transport{Base: base, Auth: auth, Cache: Default()}
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t == nil || t.Base == nil {
		return http.DefaultTransport.RoundTrip(req)
	}
	if req == nil {
		return t.Base.RoundTrip(req)
	}
	authID := ""
	if t.Auth != nil {
		authID = strings.TrimSpace(t.Auth.ID)
	}
	session := SessionFromHeader(req.Header)
	model, effort := readRequestMeta(req)

	// Always log requested effort on the way out (DSH→chat/completions path).
	logEffortDetect(authID, model, effort, usageSnapshot{}, "请求")

	if authID != "" {
		InjectIntoHeader(t.Cache, authID, model, session, req.Header)
	}

	resp, err := t.Base.RoundTrip(req)
	if err != nil || resp == nil {
		return resp, err
	}
	if authID != "" {
		CaptureFromResponse(t.Cache, authID, model, session, resp)
	}
	attachEffortDetect(resp, authID, model, effort)
	return resp, nil
}

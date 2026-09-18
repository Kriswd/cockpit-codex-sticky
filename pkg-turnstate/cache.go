package turnstate

import (
	"net/http"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	HeaderName    = "x-codex-turn-state"
	HeaderNameAlt = "X-Codex-Turn-State"
	RequiredLen   = 292
	DefaultTTL    = 30 * time.Minute
)

// Cache stores qualifying upstream turn-state tokens in process memory.
// Key = selected auth id + model + session/conversation id.
type Cache struct {
	mu      sync.Mutex
	ttl     time.Duration
	entries map[string]entry
}

type entry struct {
	value     string
	expiresAt time.Time
}

func New(ttl time.Duration) *Cache {
	if ttl <= 0 {
		ttl = DefaultTTL
	}
	return &Cache{ttl: ttl, entries: make(map[string]entry)}
}

var defaultCache = New(DefaultTTL)

func Default() *Cache { return defaultCache }

func Key(authID, model, session string) string {
	return strings.TrimSpace(authID) + "\x00" + strings.TrimSpace(model) + "\x00" + strings.TrimSpace(session)
}

func shortID(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "-"
	}
	if len(s) <= 8 {
		return s
	}
	return s[:8]
}

func NormalizeToken(values []string) (string, bool) {
	if len(values) != 1 {
		return "", false
	}
	v := strings.TrimSpace(values[0])
	if len(v) != RequiredLen {
		return "", false
	}
	return v, true
}

func (c *Cache) Put(authID, model, session, value string) bool {
	if c == nil {
		return false
	}
	token, ok := NormalizeToken([]string{value})
	if !ok || strings.TrimSpace(authID) == "" {
		return false
	}
	key := Key(authID, model, session)
	c.mu.Lock()
	defer c.mu.Unlock()
	_, replaced := c.entries[key]
	c.entries[key] = entry{value: token, expiresAt: time.Now().Add(c.ttl)}
	if replaced {
		log.Infof("[turn-state] 已更新缓存账号=%s 模型=%s 会话=%s（有效约30分钟，不打印令牌内容）", shortID(authID), blank(model), blank(session))
	} else {
		log.Infof("[turn-state] 已记住上游路由令牌 账号=%s 模型=%s 会话=%s（有效约30分钟，不打印令牌内容）", shortID(authID), blank(model), blank(session))
	}
	return true
}

func blank(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "-"
	}
	return s
}

// Get returns a cached token without extending TTL.
func (c *Cache) Get(authID, model, session string) (string, bool) {
	if c == nil || strings.TrimSpace(authID) == "" {
		return "", false
	}
	key := Key(authID, model, session)
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	ent, ok := c.entries[key]
	if !ok {
		return "", false
	}
	if now.After(ent.expiresAt) {
		delete(c.entries, key)
		return "", false
	}
	return ent.value, true
}

func SessionFromHeader(h http.Header) string {
	if h == nil {
		return ""
	}
	for _, name := range []string{"Session-Id", "Session_id", "X-Session-ID", "Thread-Id", "X-Codex-Window-Id"} {
		if v := strings.TrimSpace(h.Get(name)); v != "" {
			return v
		}
	}
	return ""
}

func HeaderHasTurnState(h http.Header) bool {
	if h == nil {
		return false
	}
	v := strings.TrimSpace(h.Get(HeaderName))
	if v == "" {
		v = strings.TrimSpace(h.Get(HeaderNameAlt))
	}
	return v != ""
}

func CaptureFromResponse(c *Cache, authID, model, session string, resp *http.Response) bool {
	if c == nil || resp == nil {
		return false
	}
	vals := resp.Header.Values(HeaderNameAlt)
	if len(vals) == 0 {
		vals = resp.Header.Values(HeaderName)
	}
	token, ok := NormalizeToken(vals)
	if !ok {
		return false
	}
	return c.Put(authID, model, session, token)
}

func InjectIntoHeader(c *Cache, authID, model, session string, dst http.Header) bool {
	if c == nil || dst == nil || HeaderHasTurnState(dst) {
		return false
	}
	token, ok := c.Get(authID, model, session)
	from := "精确会话"
	if !ok {
		token, ok = c.Get(authID, model, "")
		from = "同账号同模型兜底"
		if !ok {
			return false
		}
	}
	dst.Set(HeaderNameAlt, token)
	log.Infof("[turn-state] 已补上路由令牌（%s）账号=%s 模型=%s 会话=%s", from, shortID(authID), blank(model), blank(session))
	return true
}

// InjectFromSources copies client turn-state when present; otherwise injects cache.
func InjectFromSources(c *Cache, authID, model string, dst, ginHeaders http.Header) {
	if dst == nil {
		return
	}
	if ginHeaders != nil && !HeaderHasTurnState(dst) {
		vals := ginHeaders.Values(HeaderNameAlt)
		if len(vals) == 0 {
			vals = ginHeaders.Values(HeaderName)
		}
		if token, ok := NormalizeToken(vals); ok {
			dst.Set(HeaderNameAlt, token)
			session := SessionFromHeader(dst)
			if session == "" {
				session = SessionFromHeader(ginHeaders)
			}
			log.Infof("[turn-state] 客户端已自带路由令牌，原样转发 账号=%s 模型=%s 会话=%s", shortID(authID), blank(model), blank(session))
			_ = c.Put(authID, model, session, token)
			return
		}
	}
	session := SessionFromHeader(dst)
	if session == "" {
		session = SessionFromHeader(ginHeaders)
	}
	InjectIntoHeader(c, authID, model, session, dst)
}

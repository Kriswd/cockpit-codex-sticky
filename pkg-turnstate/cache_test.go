package turnstate

import (
	"net/http"
	"testing"
	"time"
)

func TestNormalizeAndRoundTrip(t *testing.T) {
	token := stringsRepeat("a", RequiredLen)
	c := New(time.Minute)
	if !c.Put("auth1", "gpt-test", "sess1", token) {
		t.Fatal("put failed")
	}
	got, ok := c.Get("auth1", "gpt-test", "sess1")
	if !ok || got != token {
		t.Fatalf("get = %q %v", got, ok)
	}
	c.mu.Lock()
	ent := c.entries[Key("auth1", "gpt-test", "sess1")]
	ent.expiresAt = time.Now().Add(-time.Second)
	c.entries[Key("auth1", "gpt-test", "sess1")] = ent
	c.mu.Unlock()
	if _, ok := c.Get("auth1", "gpt-test", "sess1"); ok {
		t.Fatal("expected expired miss")
	}
}

func TestRejectBadLength(t *testing.T) {
	c := New(time.Minute)
	if c.Put("auth1", "m", "s", "short") {
		t.Fatal("should reject short token")
	}
}

func TestInjectSkipsWhenPresent(t *testing.T) {
	token := stringsRepeat("b", RequiredLen)
	c := New(time.Minute)
	c.Put("auth1", "m", "s", token)
	h := http.Header{}
	h.Set(HeaderNameAlt, stringsRepeat("c", RequiredLen))
	if InjectIntoHeader(c, "auth1", "m", "s", h) {
		t.Fatal("should not overwrite existing")
	}
}

func stringsRepeat(s string, n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = s[0]
	}
	return string(b)
}

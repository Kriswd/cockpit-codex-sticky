package turnstate

import "testing"

func TestTruncationFingerprintHit(t *testing.T) {
	hits := []int64{516, 1034, 1552, 2070, 2588, 3106, 518*21 - 2}
	for _, v := range hits {
		if !TruncationFingerprintHit(v) {
			t.Fatalf("expected hit for %d", v)
		}
	}
	misses := []int64{0, 1, 515, 517, 518, 1033, 1035, 518*22 - 2, -1}
	for _, v := range misses {
		if TruncationFingerprintHit(v) {
			t.Fatalf("expected miss for %d", v)
		}
	}
}

func TestExtractClientEffort(t *testing.T) {
	cases := []struct {
		body string
		want string
	}{
		{`{"model":"m","reasoning_effort":"xhigh"}`, "xhigh"},
		{`{"model":"m","reasoning":{"effort":"high"}}`, "high"},
		{`{"model":"m","reasoning_effort":"medium","reasoning":{"effort":"low"}}`, "medium"},
		{`{"model":"m"}`, ""},
		{``, ""},
	}
	for _, c := range cases {
		got := ExtractClientEffort([]byte(c.body))
		if got != c.want {
			t.Fatalf("body=%s got=%q want=%q", c.body, got, c.want)
		}
	}
}

func TestParseUsageFromJSON(t *testing.T) {
	raw := []byte(`{"usage":{"input_tokens":10,"output_tokens":20,"total_tokens":30,"output_tokens_details":{"reasoning_tokens":516}}}`)
	u := parseUsageFromJSON(raw)
	if !u.HaveReasoning || u.ReasoningTokens != 516 || u.InputTokens != 10 || u.TotalTokens != 30 {
		t.Fatalf("unexpected %#v", u)
	}
	sse := []byte("data: {\"type\":\"response.completed\",\"response\":{\"usage\":{\"input_tokens\":1,\"output_tokens\":2,\"total_tokens\":3,\"output_tokens_details\":{\"reasoning_tokens\":1034}}}}\n\n")
	u2 := parseUsageFromSSE(sse)
	if !u2.HaveReasoning || u2.ReasoningTokens != 1034 {
		t.Fatalf("sse unexpected %#v", u2)
	}
	if fingerprintLabel(516, true) != "命中" || fingerprintLabel(100, true) != "未命中" || fingerprintLabel(0, false) != "-" {
		t.Fatal("fingerprint labels")
	}
}

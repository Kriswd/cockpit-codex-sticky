package turnstate

import (
	"bytes"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

const (
	effortDetectTag   = "[effort-detect]"
	truncationStep    = int64(518)
	truncationMaxN    = int64(21)
	sseSniffMaxBytes  = 512 * 1024
)

// ExtractClientEffort pulls client-requested reasoning effort from a request body.
// Prefer chat-completions `reasoning_effort`, else Responses `reasoning.effort`.
// Returns "" when absent (caller may log as 缺省).
func ExtractClientEffort(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	if v := strings.TrimSpace(gjson.GetBytes(body, "reasoning_effort").String()); v != "" {
		return v
	}
	if v := strings.TrimSpace(gjson.GetBytes(body, "reasoning.effort").String()); v != "" {
		return v
	}
	return ""
}

// TruncationFingerprintHit reports whether reasoning_tokens matches 518*n-2 for n=1..21.
func TruncationFingerprintHit(reasoningTokens int64) bool {
	if reasoningTokens < 516 {
		return false
	}
	if (reasoningTokens+2)%truncationStep != 0 {
		return false
	}
	n := (reasoningTokens + 2) / truncationStep
	return n >= 1 && n <= truncationMaxN
}

func effortLabel(effort string) string {
	if strings.TrimSpace(effort) == "" {
		return "缺省"
	}
	return strings.TrimSpace(effort)
}

func fingerprintLabel(reasoningTokens int64, have bool) string {
	if !have {
		return "-"
	}
	if TruncationFingerprintHit(reasoningTokens) {
		return "命中"
	}
	return "未命中"
}

type usageSnapshot struct {
	ReasoningTokens int64
	InputTokens     int64
	OutputTokens    int64
	TotalTokens     int64
	HaveReasoning   bool
}

func parseUsageFromJSON(raw []byte) usageSnapshot {
	var u usageSnapshot
	if len(raw) == 0 {
		return u
	}
	// Chat completions / Responses: usage at top level or under response.
	roots := []string{"usage", "response.usage"}
	for _, root := range roots {
		node := gjson.GetBytes(raw, root)
		if !node.Exists() {
			continue
		}
		if in := node.Get("input_tokens"); in.Exists() {
			u.InputTokens = in.Int()
		} else if in := node.Get("prompt_tokens"); in.Exists() {
			u.InputTokens = in.Int()
		}
		if out := node.Get("output_tokens"); out.Exists() {
			u.OutputTokens = out.Int()
		} else if out := node.Get("completion_tokens"); out.Exists() {
			u.OutputTokens = out.Int()
		}
		if tot := node.Get("total_tokens"); tot.Exists() {
			u.TotalTokens = tot.Int()
		}
		rt := node.Get("output_tokens_details.reasoning_tokens")
		if !rt.Exists() {
			rt = node.Get("completion_tokens_details.reasoning_tokens")
		}
		if rt.Exists() {
			u.ReasoningTokens = rt.Int()
			u.HaveReasoning = true
		}
		return u
	}
	return u
}

// parseUsageFromSSE scans SSE/text for the last usage object containing reasoning_tokens.
func parseUsageFromSSE(buf []byte) usageSnapshot {
	if len(buf) == 0 {
		return usageSnapshot{}
	}
	// Prefer last occurrence of reasoning_tokens then walk outward for a JSON object.
	needle := []byte(`"reasoning_tokens"`)
	idx := bytes.LastIndex(buf, needle)
	if idx < 0 {
		return usageSnapshot{}
	}
	// Expand left to nearest '{' that yields a parseable usage-ish blob.
	start := idx
	for start > 0 && buf[start] != '{' {
		start--
	}
	end := idx
	for end < len(buf) && buf[end] != '}' {
		end++
	}
	if end < len(buf) {
		end++
	}
	// Also try wrapping larger windows upward to include parent "usage".
	candidates := [][]byte{buf[start:end]}
	for up := 0; up < 3; up++ {
		p := start
		for p > 0 {
			p--
			if buf[p] == '{' {
				candidates = append(candidates, buf[p:end])
				start = p
				break
			}
		}
	}
	var best usageSnapshot
	for i := len(candidates) - 1; i >= 0; i-- {
		u := parseUsageFromJSON(candidates[i])
		if u.HaveReasoning {
			return u
		}
		if u.TotalTokens > 0 || u.InputTokens > 0 || u.OutputTokens > 0 {
			best = u
		}
		// Try wrapping as {"usage": ...}
		wrapped := append(append([]byte(`{"usage":`), candidates[i]...), '}')
		u2 := parseUsageFromJSON(wrapped)
		if u2.HaveReasoning {
			return u2
		}
	}
	// Fallback: whole buffer as JSON (non-SSE edge)
	if u := parseUsageFromJSON(buf); u.HaveReasoning || u.TotalTokens > 0 {
		return u
	}
	return best
}

func logEffortDetect(authID, model, effort string, usage usageSnapshot, phase string) {
	// Request line: short outbound note. Response line matches:
	// [effort-detect] 请求档位=xhigh 模型=gpt-6-astra 推理token=516 截断指纹=命中
	parts := []string{effortDetectTag}
	parts = append(parts,
		"请求档位="+effortLabel(effort),
		"模型="+blank(model),
	)
	if phase == "请求" {
		parts = append(parts, "账号="+shortID(authID))
		log.Infof("%s", strings.Join(parts, " "))
		return
	}
	if usage.HaveReasoning {
		parts = append(parts, "推理token="+strconv.FormatInt(usage.ReasoningTokens, 10))
	} else {
		parts = append(parts, "推理token=-")
	}
	if usage.InputTokens > 0 {
		parts = append(parts, "input="+strconv.FormatInt(usage.InputTokens, 10))
	}
	if usage.OutputTokens > 0 {
		parts = append(parts, "output="+strconv.FormatInt(usage.OutputTokens, 10))
	}
	if usage.TotalTokens > 0 {
		parts = append(parts, "total="+strconv.FormatInt(usage.TotalTokens, 10))
	}
	parts = append(parts,
		"截断指纹="+fingerprintLabel(usage.ReasoningTokens, usage.HaveReasoning),
		"账号="+shortID(authID),
	)
	log.Infof("%s", strings.Join(parts, " "))
}

// readRequestMeta reads JSON body once, restores it, returns model + client effort.
func readRequestMeta(req *http.Request) (model, effort string) {
	if req == nil || req.Body == nil || req.Body == http.NoBody {
		return "", ""
	}
	ct := strings.ToLower(req.Header.Get("Content-Type"))
	if ct != "" && !strings.Contains(ct, "json") {
		return "", ""
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		req.Body = io.NopCloser(bytes.NewReader(nil))
		return "", ""
	}
	req.Body = io.NopCloser(bytes.NewReader(body))
	model = strings.TrimSpace(gjson.GetBytes(body, "model").String())
	effort = ExtractClientEffort(body)
	return model, effort
}

func attachEffortDetect(resp *http.Response, authID, model, effort string) {
	if resp == nil || resp.Body == nil {
		logEffortDetect(authID, model, effort, usageSnapshot{}, "响应")
		return
	}
	ct := strings.ToLower(resp.Header.Get("Content-Type"))
	isSSE := strings.Contains(ct, "text/event-stream") || strings.Contains(ct, "event-stream")
	if !isSSE && strings.Contains(ct, "json") {
		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			resp.Body = io.NopCloser(bytes.NewReader(nil))
			logEffortDetect(authID, model, effort, usageSnapshot{}, "响应")
			return
		}
		resp.Body = io.NopCloser(bytes.NewReader(body))
		u := parseUsageFromJSON(body)
		logEffortDetect(authID, model, effort, u, "响应")
		return
	}
	// SSE / unknown: tee-sniff until EOF (pragmatic; usage usually at stream end).
	resp.Body = &effortSniffBody{
		ReadCloser: resp.Body,
		authID:     authID,
		model:      model,
		effort:     effort,
		isSSE:      isSSE || ct == "",
	}
}

type effortSniffBody struct {
	io.ReadCloser
	authID, model, effort string
	isSSE                 bool
	mu                    sync.Mutex
	buf                   bytes.Buffer
	logged                bool
}

func (e *effortSniffBody) Read(p []byte) (int, error) {
	n, err := e.ReadCloser.Read(p)
	if n > 0 {
		e.mu.Lock()
		if e.buf.Len() < sseSniffMaxBytes {
			remain := sseSniffMaxBytes - e.buf.Len()
			if n < remain {
				remain = n
			}
			_, _ = e.buf.Write(p[:remain])
		}
		e.mu.Unlock()
	}
	if err == io.EOF {
		e.flush()
	}
	return n, err
}

func (e *effortSniffBody) Close() error {
	e.flush()
	return e.ReadCloser.Close()
}

func (e *effortSniffBody) flush() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.logged {
		return
	}
	e.logged = true
	raw := e.buf.Bytes()
	var u usageSnapshot
	if e.isSSE {
		u = parseUsageFromSSE(raw)
	} else {
		u = parseUsageFromJSON(raw)
		if !u.HaveReasoning {
			u = parseUsageFromSSE(raw)
		}
	}
	logEffortDetect(e.authID, e.model, e.effort, u, "响应")
}

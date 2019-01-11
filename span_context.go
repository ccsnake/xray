package xray

import (
	"bytes"
	"github.com/opentracing/opentracing-go"
	"strings"
)

const (
	// RootPrefix is the prefix for
	// Root attribute in X-Amzn-Trace-Id.
	RootPrefix = "Root="

	// ParentPrefix is the prefix for
	// Parent attribute in X-Amzn-Trace-Id.
	ParentPrefix = "Parent="

	// SampledPrefix is the prefix for
	// Sampled attribute in X-Amzn-Trace-Id.
	SampledPrefix = "Sampled="

	// SelfPrefix is the prefix for
	// Self attribute in X-Amzn-Trace-Id.
	SelfPrefix = "Self="
)

type SpanContext struct {
	// A probabilistically unique identifier for a [multi-span] trace.
	TraceID string

	// The SpanID of this Context's parent, or nil if there is no parent.
	ParentID string

	SamplingDecision string

	// A probabilistically unique identifier for a span.
	SpanID string

	// The span's associated baggage.
	AdditionalData map[string]string // initialized on first use
}

// ForeachBaggageItem xray not support baggage item
func (sc *SpanContext) ForeachBaggageItem(handler func(k, v string) bool) {
	return
}


func (sc *SpanContext) Inject(h opentracing.TextMapWriter) {
	var p [][]byte
	if sc.TraceID != "" {
		p = append(p, []byte(RootPrefix+sc.TraceID))
	}
	if sc.ParentID != "" {
		p = append(p, []byte(ParentPrefix+sc.SpanID))
	}
	p = append(p, []byte(sc.SamplingDecision))
	for key := range sc.AdditionalData {
		p = append(p, []byte(key+"="+sc.AdditionalData[key]))
	}
	h.Set("X-Amzn-Trace-Id", string(bytes.Join(p, []byte(";"))))
}

func Extract(s string) (*SpanContext, error) {
	if len(s) == 0 {
		return nil, opentracing.ErrSpanContextNotFound
	}
	ret := &SpanContext{
		SamplingDecision: "unknown",
		AdditionalData:   make(map[string]string),
	}
	parts := strings.Split(s, ";")
	for i := range parts {
		p := strings.TrimSpace(parts[i])
		value, valid := valueFromKeyValuePair(p)
		if valid {
			if strings.HasPrefix(p, RootPrefix) {
				ret.TraceID = value
			} else if strings.HasPrefix(p, ParentPrefix) {
				ret.SpanID = value
			} else if strings.HasPrefix(p, SampledPrefix) {
				ret.SamplingDecision = p
			} else if !strings.HasPrefix(p, SelfPrefix) {
				key, valid := keyFromKeyValuePair(p)
				if valid {
					ret.AdditionalData[key] = value
				}
			}
		}
	}

	return ret, nil
}

func keyFromKeyValuePair(s string) (string, bool) {
	e := strings.Index(s, "=")
	if -1 != e {
		return s[:e], true
	}
	return "", false
}

func valueFromKeyValuePair(s string) (string, bool) {
	e := strings.Index(s, "=")
	if -1 != e {
		return s[e+1:], true
	}
	return "", false
}

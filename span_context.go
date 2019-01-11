package xray

import (
	"bytes"
	"github.com/opentracing/opentracing-go"
	"net/http"
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
	ParentSpanID string

	SamplingDecision string

	// A probabilistically unique identifier for a span.
	SpanID string

	// The span's associated baggage.
	Baggage map[string]string // initialized on first use
}

func (sc *SpanContext) ForeachBaggageItem(handler func(k, v string) bool) {
	if sc.Baggage == nil {
		return
	}

	for key, value := range sc.Baggage {
		handler(key, value)
	}
}

func (sc *SpanContext) String() string {
	var p [][]byte
	if sc.TraceID != "" {
		p = append(p, []byte(RootPrefix+sc.TraceID))
	}
	if sc.ParentSpanID != "" {
		p = append(p, []byte(ParentPrefix+sc.ParentSpanID))
	}
	p = append(p, []byte(sc.SamplingDecision))
	for key := range sc.Baggage {
		p = append(p, []byte(key+"="+sc.Baggage[key]))
	}
	return string(bytes.Join(p, []byte(";")))
}

func (sc *SpanContext) Inject(h http.Header) {
	h.Set("X-Amzn-Trace-Id", sc.String())
}

func Extract(s string) (*SpanContext, error) {
	if len(s) == 0 {
		return nil, opentracing.ErrSpanContextNotFound
	}
	ret := &SpanContext{
		SamplingDecision: "unknown",
		Baggage:   make(map[string]string),
	}
	parts := strings.Split(s, ";")
	for i := range parts {
		p := strings.TrimSpace(parts[i])
		value, valid := valueFromKeyValuePair(p)
		if valid {
			if strings.HasPrefix(p, RootPrefix) {
				ret.TraceID = value
			} else if strings.HasPrefix(p, ParentPrefix) {
				ret.ParentSpanID = value
			} else if strings.HasPrefix(p, SampledPrefix) {
				ret.SamplingDecision = p
			} else if !strings.HasPrefix(p, SelfPrefix) {
				key, valid := keyFromKeyValuePair(p)
				if valid {
					ret.Baggage[key] = value
				}
			}
		}
	}
	return ret,nil
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


package xray

import "github.com/opentracing/opentracing-go"

const (
	AMZHeader = "X-Amzn-Trace-Id"
)

type textPropagator struct {
	*Tracer
}

func (t *textPropagator) Inject(sc *SpanContext, carrier opentracing.TextMapWriter) {
	sc.Inject(carrier)
}

func (t *textPropagator) Extract(carrier opentracing.TextMapReader) (sc opentracing.SpanContext, err error) {
	carrier.ForeachKey(func(key, val string) error {
		if key == AMZHeader {
			sc, err = Extract(val)
			return opentracing.ErrSpanContextCorrupted
		}
		return nil
	})


	return
}

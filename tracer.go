package xray

import (
	"errors"
	"fmt"
	"github.com/opentracing/opentracing-go"
	"math/rand"
	"sync"
	"time"
)

var ErrNotXRaySpanContext = errors.New("not a xray span context")

// NewTraceID generates a string format of random trace ID.
func NewTraceID() string {
	var r [12]byte
	_, err := rand.Read(r[:])
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("1-%08x-%02x", time.Now().Unix(), r)
}

// NewSegmentID generates a string format of segment ID.
func NewSegmentID() string {
	var r [8]byte
	_, err := rand.Read(r[:])
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%02x", r)
}

type Tracer struct {
	mu sync.Mutex

	collector *UDPCollector
	textPropagator

	serviceName string
}

func (t *Tracer) StartSpan(operationName string, opts ...opentracing.StartSpanOption) opentracing.Span {
	var option opentracing.StartSpanOptions
	for _, opt := range opts {
		opt.Apply(&option)
	}

	now := time.Now()
	sp := &Span{
		Name:      operationName,
		ID:        NewSegmentID(),
		StartTime: float64(now.UnixNano()) / float64(time.Second),
		TraceID:   NewTraceID(),
		start:     now,
		tracer:    t,
	}

	for _, ref := range option.References {
		switch ref.Type {
		case opentracing.ChildOfRef:
			sc, ok := ref.ReferencedContext.(*SpanContext)
			if !ok {
				continue
			}

			sp.TraceID = sc.TraceID
			if len(sc.SpanID) > 0 {
				sp.ParentSpanID = sc.SpanID
				sp.Type = "subsegment"
			}
		case opentracing.FollowsFromRef:
			fmt.Println("bingo....")
		default:
		}
	}

	return sp
}

func (t *Tracer) Inject(sm opentracing.SpanContext, format interface{}, carrier interface{}) error {
	sc, ok := sm.(*SpanContext)
	if !ok {
		return ErrNotXRaySpanContext
	}

	switch format {
	case opentracing.TextMap, opentracing.HTTPHeaders:
		carrier, ok := carrier.(opentracing.TextMapWriter)
		if !ok {
			return opentracing.ErrInvalidCarrier
		}
		t.textPropagator.Inject(sc, carrier)
		return nil
	default:
		return opentracing.ErrUnsupportedFormat
	}
}

func (t *Tracer) Extract(format interface{}, carrier interface{}) (opentracing.SpanContext, error) {
	switch format {
	case opentracing.TextMap, opentracing.HTTPHeaders:
		carrier, ok := carrier.(opentracing.TextMapReader)
		if !ok {
			return nil, opentracing.ErrInvalidCarrier
		}
		return t.textPropagator.Extract(carrier)
	default:
		return nil, opentracing.ErrUnsupportedFormat
	}
}

func (t *Tracer) finish(sp *Span) {
	t.collector.Send(sp)
}

func NewTracer(addr string, serviceName string) *Tracer {
	return &Tracer{
		collector:   NewUDPCollector(addr),
		serviceName: serviceName,
	}
}

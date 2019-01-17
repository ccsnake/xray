package xray

import (
	"encoding/json"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
	"sync"
	"time"
)

type Span struct {
	mu sync.RWMutex

	Name             string  `json:"name"`
	ID               string  `json:"id"`
	StartTime        float64 `json:"start_time"`
	EndTime          float64 `json:"end_time,omitempty"`
	InProgress       bool    `json:"in_progress"`
	TraceID          string  `json:"trace_id"`
	Type             string  `json:"type,omitempty"`
	ParentSpanID     string  `json:"parent_id,omitempty"`
	SamplingDecision string  `json:"sampling_decision,omitempty"`
	Annotations      map[string]interface{}
	MetaData         map[string]interface{} `json:"meta_data,omitempty"`

	start  time.Time
	tracer *Tracer
}

func (sp *Span) Finish() {
	sp.mu.Lock()
	sp.EndTime = float64(time.Now().UnixNano()) / float64(time.Second)
	sp.InProgress = false
	sp.mu.Unlock()

	sp.tracer.finish(sp)
}

func (sp *Span) FinishWithOptions(opts opentracing.FinishOptions) {
	finishTime := opts.FinishTime
	if finishTime.IsZero() {
		finishTime = time.Now()
	}
	sp.mu.Lock()
	sp.EndTime = float64(finishTime.UnixNano()) / float64(time.Second)
	sp.InProgress = false
	sp.mu.Unlock()

	sp.tracer.finish(sp)
}

func (sp *Span) Context() opentracing.SpanContext {
	sp.mu.RLock()
	defer sp.mu.RUnlock()
	
	// todo add ann & metadata
	return &SpanContext{
		TraceID:          sp.TraceID,
		ParentID:         sp.ParentSpanID,
		SamplingDecision: sp.SamplingDecision,
		SpanID:           sp.ID,
		AdditionalData:   nil,
	}
}

func (sp *Span) SetOperationName(operationName string) opentracing.Span {
	sp.mu.Lock()
	sp.Name = operationName
	sp.mu.Unlock()

	return sp
}

func (sp *Span) SetTag(key string, value interface{}) opentracing.Span {
	sp.mu.Lock()
	if sp.Annotations == nil {
		sp.Annotations = make(map[string]interface{})
	}
	sp.Annotations[key] = value
	sp.mu.Unlock()

	return sp
}

func (sp *Span) LogFields(fields ...log.Field) {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	if sp.MetaData == nil {
		sp.MetaData = make(map[string]interface{})
	}

	for _, f := range fields {
		sp.MetaData[f.Key()] = f.Value()
	}
}

func (sp *Span) LogKV(alternatingKeyValues ...interface{}) {
	fields, err := log.InterleavedKVToFields(alternatingKeyValues...)
	if err != nil {
		sp.LogFields(log.Error(err), log.String("function", "LogKV"))
		return
	}
	sp.LogFields(fields...)
}

// SetBaggageItem not supported now
func (sp *Span) SetBaggageItem(restrictedKey, value string) opentracing.Span {
	return sp
}

// BaggageItem not supported now
func (sp *Span) BaggageItem(restrictedKey string) string {
	return ""
}

func (sp *Span) Tracer() opentracing.Tracer {
	return sp.tracer
}

func (sp *Span) LogEvent(event string) {
	sp.LogFields(log.Bool("event."+event, true))
}

func (sp *Span) LogEventWithPayload(event string, payload interface{}) {
	sp.LogFields(log.Object("event."+event, payload))
}

func (sp *Span) Log(data opentracing.LogData) {
	sp.LogFields(log.Object("event."+data.Event, data.Payload))
}

func (sp *Span) Encode() ([]byte, error) {
	sp.mu.Lock()
	sg := Segment{
		Name:        fixSegmentName(sp.Name),
		ID:          sp.ID,
		StartTime:   sp.StartTime,
		EndTime:     sp.EndTime,
		TraceID:     sp.TraceID,
		Type:        sp.Type,
		ParentID:    sp.ParentSpanID,
		Service:     sp.tracer.serviceName,
		Annotations: sp.Annotations,
		MetaData:    sp.MetaData,
		Namespace:   "remote",
	}

	var hi httpInfo
	for key, value := range sp.Annotations {
		switch key {
		case "http.method":
			if method, ok := value.(string); ok {
				hi.Method = method
			}
			sg.Http = &hi
		case "http.status_code":
			state, ok := value.(int)
			if ok {
				hi.Response.Status = state
			}
			sg.Http = &hi
			if state == 429 {
				sg.Throttle = true
			} else if state/100 == 5 {
				sg.Fault = true
			} else if state/100 == 4 {
				sg.Error = true
			}
		case "http.url":
			if u, ok := value.(string); ok {
				hi.URL = u
			}
			sg.Http = &hi
		case "error":
			if u, ok := value.(string); ok {
				hi.URL = u
			}
			sg.Error = true
		}
	}

	if sp.Annotations != nil {
		if val, ok := sp.Annotations["error"]; ok && !sg.Throttle && !sg.Fault {
			if err, ok := val.(bool); ok {
				sg.Error = err
			}
		}
	}

	sp.mu.Unlock()

	return json.Marshal(sg)
}

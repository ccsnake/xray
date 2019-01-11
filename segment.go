package xray

import "regexp"

var (
	// reInvalidSpanCharacters defines the invalid letters in a span name as per
	// https://docs.aws.amazon.com/xray/latest/devguide/xray-api-segmentdocuments.html
	reInvalidSpanCharacters = regexp.MustCompile(`[^ 0-9\p{L}N_.:/%&#=+,\-@]`)
	// reInvalidAnnotationCharacters defines the invalid letters in an annotation key as per
	// https://docs.aws.amazon.com/xray/latest/devguide/xray-api-segmentdocuments.html
	reInvalidAnnotationCharacters = regexp.MustCompile(`[^a-zA-Z0-9_]`)

	// defaultSpanName will be used if there are no valid xray characters in the
	// span name
	defaultSegmentName = "span"

	// maxSegmentNameLength the maximum length of a segment name
	maxSegmentNameLength = 200
)

type Segment struct {
	Name       string  `json:"name"`
	ID         string  `json:"id"`
	StartTime  float64 `json:"start_time"`
	EndTime    float64 `json:"end_time,omitempty"`
	InProgress bool    `json:"in_progress"`

	TraceID     string                 `json:"trace_id"`
	Type        string                 `json:"type,omitempty"`
	ParentID    string                 `json:"parent_id,omitempty"`
	Service     string                 `json:"service,omitempty"`
	Namespace   string                 `json:"namespace,omitempty"` // must be remote
	MetaData    map[string]interface{} `json:"meta_data,omitempty"`
	Annotations map[string]interface{} `json:"annotations,omitempty"`
}

func fixSegmentName(name string) string {
	if reInvalidSpanCharacters.MatchString(name) {
		// only allocate for ReplaceAllString if we need to
		name = reInvalidSpanCharacters.ReplaceAllString(name, "")
	}

	if length := len(name); length > maxSegmentNameLength {
		name = name[0:maxSegmentNameLength]
	} else if length == 0 {
		name = defaultSegmentName
	}

	return name
}

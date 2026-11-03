package observability

import (
	"net/http"
	"regexp"
	"strings"
)

// TraceKey, Cloud Logging'in istekleri izlerle eşleştirdiği alan.
const TraceKey = "logging.googleapis.com/trace"

var (
	traceparentRE = regexp.MustCompile(`^[0-9a-f]{2}-([0-9a-f]{32})-[0-9a-f]{16}-[0-9a-f]{2}$`)
	cloudTraceRE  = regexp.MustCompile(`^([0-9a-fA-F]{32})(/.*)?$`)
)

// TraceID, istekteki izleme kimliğini döndürür (yoksa "").
// Önce W3C `traceparent`, sonra Cloud Run'ın `X-Cloud-Trace-Context`
// başlığına bakılır.
func TraceID(r *http.Request) string {
	if m := traceparentRE.FindStringSubmatch(r.Header.Get("traceparent")); m != nil {
		if m[1] != strings.Repeat("0", 32) {
			return m[1]
		}
	}
	if m := cloudTraceRE.FindStringSubmatch(r.Header.Get("X-Cloud-Trace-Context")); m != nil {
		return strings.ToLower(m[1])
	}
	return ""
}

// TraceResource, Cloud Logging izleme alanının değerini döndürür.
// Proje kimliği bilinmiyorsa "" (alan yazılmaz).
func TraceResource(projectID, traceID string) string {
	if projectID == "" || traceID == "" {
		return ""
	}
	return "projects/" + projectID + "/traces/" + traceID
}

package handler

import (
	"net"
	"net/http"
	"time"

	"github.com/OneDayX/go-metrics/internal/audit"
	"github.com/OneDayX/go-metrics/internal/models"
)

// audit sends an audit event about the saved metrics.
func (h *Handler) audit(r *http.Request, metrics ...models.Metric) {
	if h.auditor == nil {
		return
	}

	names := make([]string, 0, len(metrics))
	for _, m := range metrics {
		names = append(names, m.ID)
	}

	// RemoteAddr looks like "ip:port", we need only the ip
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}

	h.auditor.Publish(audit.Event{
		TS:        time.Now().Unix(),
		Metrics:   names,
		IPAddress: ip,
	})
}

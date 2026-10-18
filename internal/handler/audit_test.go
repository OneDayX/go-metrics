package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/OneDayX/go-metrics/internal/audit"
	"github.com/OneDayX/go-metrics/internal/repository"
	"github.com/OneDayX/go-metrics/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testAuditor struct {
	events []audit.Event
}

func (a *testAuditor) Publish(event audit.Event) {
	a.events = append(a.events, event)
}

func TestUpdatesJSONAudit(t *testing.T) {
	auditor := &testAuditor{}
	svc := service.NewMetricService(repository.NewMemStorage())
	h := NewHandler(nil, auditor)

	body := `[{"id":"Alloc","type":"gauge","value":1.5},{"id":"PollCount","type":"counter","delta":1}]`
	req := httptest.NewRequest(http.MethodPost, "/updates/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "192.168.0.42:12345"
	w := httptest.NewRecorder()

	h.UpdatesJSON(svc)(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Len(t, auditor.events, 1)
	assert.Equal(t, []string{"Alloc", "PollCount"}, auditor.events[0].Metrics)
	assert.Equal(t, "192.168.0.42", auditor.events[0].IPAddress)
}

func TestUpdatesJSONNoAuditOnError(t *testing.T) {
	auditor := &testAuditor{}
	svc := service.NewMetricService(repository.NewMemStorage())
	h := NewHandler(nil, auditor)

	req := httptest.NewRequest(http.MethodPost, "/updates/", strings.NewReader(`not a json`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.UpdatesJSON(svc)(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, auditor.events)
}

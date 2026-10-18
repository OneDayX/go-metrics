package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/OneDayX/go-metrics/internal/handler"
	"github.com/OneDayX/go-metrics/internal/models"
	"github.com/OneDayX/go-metrics/internal/repository"
	"github.com/OneDayX/go-metrics/internal/service"
	"github.com/go-chi/chi/v5"
)

// Save a metric with the type, name and value in the URL.
func ExampleHandler_Update() {
	svc := service.NewMetricService(repository.NewMemStorage())
	h := handler.NewHandler(nil, nil)

	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", h.Update(svc))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/update/counter/PollCount/5", nil))
	fmt.Println(w.Code)

	// A counter value must be an integer.
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/update/counter/PollCount/1.5", nil))
	fmt.Println(w.Code)

	// Output:
	// 200
	// 400
}

// Get the value of a metric as plain text.
func ExampleHandler_Get() {
	svc := service.NewMetricService(repository.NewMemStorage())
	h := handler.NewHandler(nil, nil)

	r := chi.NewRouter()
	r.Get("/value/{type}/{name}", h.Get(svc))

	_ = svc.Update(context.Background(), models.Metric{
		ID: "Alloc", MType: models.MetricTypeGauge, Value: models.Ptr(12.5),
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/value/gauge/Alloc", nil))
	fmt.Println(w.Code, w.Body.String())

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/value/gauge/Unknown", nil))
	fmt.Println(w.Code)

	// Output:
	// 200 12.5
	// 404
}

// Get an HTML page with all metrics.
func ExampleHandler_List() {
	svc := service.NewMetricService(repository.NewMemStorage())
	h := handler.NewHandler(nil, nil)

	r := chi.NewRouter()
	r.Get("/", h.List(svc))

	_ = svc.Update(context.Background(), models.Metric{
		ID: "Alloc", MType: models.MetricTypeGauge, Value: models.Ptr(12.5),
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	fmt.Println(w.Code, w.Header().Get("Content-Type"))
	fmt.Println(strings.Contains(w.Body.String(), "<td>Alloc</td>"))

	// Output:
	// 200 text/html; charset=utf-8
	// true
}

// Check the database connection. Without a database it still answers 200.
func ExampleHandler_Ping() {
	h := handler.NewHandler(nil, nil)

	r := chi.NewRouter()
	r.Get("/ping", h.Ping(nil))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ping", nil))
	fmt.Println(w.Code)

	// Output:
	// 200
}

// Save one metric sent as JSON.
func ExampleHandler_UpdateJSON() {
	svc := service.NewMetricService(repository.NewMemStorage())
	h := handler.NewHandler(nil, nil)

	r := chi.NewRouter()
	r.Post("/update", h.UpdateJSON(svc))

	body := `{"id":"PollCount","type":"counter","delta":5}`

	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	fmt.Println(w.Code)

	// Without Content-Type: application/json the request is rejected.
	req = httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(body))
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	fmt.Println(w.Code)

	// Output:
	// 200
	// 400
}

// Save a batch of metrics sent as a JSON array, the way the agent does.
func ExampleHandler_UpdatesJSON() {
	svc := service.NewMetricService(repository.NewMemStorage())
	h := handler.NewHandler(nil, nil)

	r := chi.NewRouter()
	r.Post("/updates", h.UpdatesJSON(svc))

	body := `[
		{"id":"Alloc","type":"gauge","value":12.5},
		{"id":"PollCount","type":"counter","delta":5},
		{"id":"PollCount","type":"counter","delta":3}
	]`

	req := httptest.NewRequest(http.MethodPost, "/updates", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	fmt.Println(w.Code)

	// The counter deltas are added up.
	pollCount, _ := svc.Fetch(context.Background(), "PollCount")
	fmt.Println(*pollCount.Delta)

	// Output:
	// 200
	// 8
}

// Get a metric as JSON: send its name and type, get it back with the value.
func ExampleHandler_ValueJSON() {
	svc := service.NewMetricService(repository.NewMemStorage())
	h := handler.NewHandler(nil, nil)

	r := chi.NewRouter()
	r.Post("/value", h.ValueJSON(svc))

	_ = svc.Update(context.Background(), models.Metric{
		ID: "Alloc", MType: models.MetricTypeGauge, Value: models.Ptr(12.5),
	})

	req := httptest.NewRequest(http.MethodPost, "/value", strings.NewReader(`{"id":"Alloc","type":"gauge"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	fmt.Println(w.Code, w.Body.String())

	// Output:
	// 200 {"id":"Alloc","type":"gauge","value":12.5}
}

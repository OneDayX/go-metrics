package handler

import (
	"html/template"
	"net/http"

	"github.com/OneDayX/go-metrics/internal/models"
	"go.uber.org/zap"
)

const listTpl = `
<!DOCTYPE html>
<html>
	<head>
		<meta charset="UTF-8">
		<title>Metrics List</title>
	</head>
	<body>
		<table>
		  {{range .Metrics}}
				<tr>
					<td>{{ .ID }}</td>
					<td>{{ .Value }}</td>
				</tr>
			{{end}}
		</table>
	</body>
</html>`

type metricLister interface {
	FetchAll() []models.Metric
}

// List returns an HTTP handler that renders all stored metrics.
func (h *Handler) List(svc metricLister) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t, err := template.New("homepage").Parse(listTpl)
		if err != nil {
			h.log.Error("failed to parse metrics list template",
				zap.String("uri", r.RequestURI),
				zap.Error(err),
			)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		metrics := struct {
			Metrics []models.Metric
		}{
			Metrics: svc.FetchAll(),
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		err = t.Execute(w, metrics)

		if err != nil {
			h.log.Error("failed to execute metrics list template",
				zap.String("uri", r.RequestURI),
				zap.Error(err),
			)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

	}
}

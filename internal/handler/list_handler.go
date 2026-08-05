package handler

import (
	"html/template"
	"net/http"

	"github.com/OneDayX/go-metrics/internal/models"
	"github.com/OneDayX/go-metrics/internal/server/middleware"
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

func ListHandler(svc metricLister) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := middleware.LoggerFromContext(r.Context())

		t, err := template.New("homepage").Parse(listTpl)
		if err != nil {
			log.Error("failed to parse metrics list template",
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

		err = t.Execute(w, metrics)

		if err != nil {
			log.Error("failed to execute metrics list template",
				zap.String("uri", r.RequestURI),
				zap.Error(err),
			)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

	}
}

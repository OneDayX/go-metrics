package handler

import (
	"net/http"
	"text/template"

	"github.com/OneDayX/go-metrics/internal/models"
	"github.com/OneDayX/go-metrics/internal/service"
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

func ListHandler(svc *service.MetricService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t, err := template.New("homepage").Parse(listTpl)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		metrics := struct {
			Metrics []models.Metric
		}{
			Metrics: svc.FetchAll(),
		}

		err = t.Execute(w, metrics)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

	}
}

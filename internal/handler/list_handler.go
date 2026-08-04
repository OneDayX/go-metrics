package handler

import (
	"html/template"
	"net/http"

	"github.com/OneDayX/go-metrics/internal/models"
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
		t, err := template.New("homepage").Parse(listTpl)
		if err != nil {
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
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

	}
}

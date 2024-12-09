package main

import (
	"memstorage"
	"metrics"
	"net/http"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	storage := &memstorage.MemStorage{
		Gauges:   make(map[string]metrics.Gauge),
		Counters: make(map[string]metrics.Counter),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/update/{type}/{name}/{value}", updateHander(storage))

	return http.ListenAndServe(`:8080`, mux)
}

func updateHander(storage *memstorage.MemStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// if r.PathValue("type")
		w.WriteHeader(http.StatusOK)
	}
}

// func webhook(w http.ResponseWriter, r *http.Request) {
// 	if r.Method != http.MethodPost {
// 		w.WriteHeader(http.StatusMethodNotAllowed)
// 		return
// 	}

// 	w.Header().Set("Content-Type", "application/json")
// 	_, _ = w.Write([]byte(`
//       {
//         "response": {
//           "text": "Извините, я пока ничего не умею"
//         },
//         "version": "1.0"
//       }
//     `))
// }

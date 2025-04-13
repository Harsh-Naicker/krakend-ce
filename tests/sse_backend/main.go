package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Event struct {
	ID    string `json:"id"`
	Data  string `json:"data"`
	Event string `json:"event"`
}

func main() {
	http.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		// Set SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		// Create a channel to send events
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
			return
		}

		// Send events every second
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for i := 0; i < 5; i++ {
			select {
			case <-ticker.C:
				event := Event{
					ID:    fmt.Sprintf("%d", i),
					Data:  fmt.Sprintf("Event %d", i),
					Event: "test",
				}
				data, _ := json.Marshal(event)
				fmt.Fprintf(w, "id: %s\nevent: %s\ndata: %s\n\n", event.ID, event.Event, data)
				flusher.Flush()
			case <-r.Context().Done():
				return
			}
		}
	})

	fmt.Println("SSE backend server running on :8081")
	http.ListenAndServe(":8081", nil)
}

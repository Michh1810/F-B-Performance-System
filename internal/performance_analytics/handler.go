// this handler will Receive a Web Request: Capture the incoming HTTP request from the browser/frontend.

// Send a Web Response: Convert your Go data structures into raw JSON text and stream it back across the internet.

package performance_analytics

import (
	"encoding/json"
	"net/http"
)

type Handler struct{ // this is called an anchor struct that will hold all dependencies, like a parent that have skills
	s *Service
}

func NewHandler(s *Service) *Handler{
	return &Handler{s: s}
} 
// Core web method
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request){
	if r.Method != http.MethodGet{
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	// Fetch the contract data from service file
	data := h.s.GetDashBoardData()
	// this will set the instruction header for browser know what format the data sending is application/json
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // confirm data is good to go
	// Convert data into raw JSON text and send it back
	json.NewEncoder(w).Encode(data)
}

package demand_forecast

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	defer r.Body.Close()

	var request ForecastBatchRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if err := ensureSingleJSONValue(decoder); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateForecastRequests(request.Items); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	response, err := h.service.ForecastMenuItems(r.Context(), request.Items)
	if err != nil {
		if errors.Is(err, ErrNoHistoricalTransactions) {
			writeJSONError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "unable to forecast menu items")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)

func validateForecastRequests(items []ForecastRequest) error {
	if len(items) == 0 {
		return fmt.Errorf("items must contain at least one forecast request")
	}

	for i, item := range items {
		fieldPrefix := fmt.Sprintf("items[%d]", i)
		if strings.TrimSpace(item.ItemID) == "" {
			return fmt.Errorf("%s.item_id is required", fieldPrefix)
		}
		if !uuidPattern.MatchString(item.ItemID) {
			return fmt.Errorf("%s.item_id must be a UUID", fieldPrefix)
		}
		if strings.TrimSpace(item.ItemName) == "" {
			return fmt.Errorf("%s.item_name is required", fieldPrefix)
		}
		if item.ForecastHorizonDays <= 0 {
			return fmt.Errorf("%s.forecast_horizon_days must be greater than zero", fieldPrefix)
		}
		if item.PriceCents <= 0 {
			return fmt.Errorf("%s.price_cents must be greater than zero", fieldPrefix)
		}
		if item.EstimatedCOGSCents < 0 {
			return fmt.Errorf("%s.estimated_cogs_cents cannot be negative", fieldPrefix)
		}
		for j, transaction := range item.HistoricalTransactions {
			transactionPrefix := fmt.Sprintf("%s.historical_transactions[%d]", fieldPrefix, j)
			if transaction.Timestamp.IsZero() {
				return fmt.Errorf("%s.timestamp is required and must be RFC3339", transactionPrefix)
			}
			if transaction.Quantity < 0 {
				return fmt.Errorf("%s.quantity cannot be negative", transactionPrefix)
			}
		}
	}
	return nil
}

func ensureSingleJSONValue(decoder *json.Decoder) error {
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return fmt.Errorf("request body must contain exactly one JSON object")
		}
		return fmt.Errorf("invalid request body: %w", err)
	}
	return nil
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

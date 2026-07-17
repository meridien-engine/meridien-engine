package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/meridien-engine/meridien-engine/internal/db"
	"github.com/meridien-engine/meridien-engine/internal/erp"
	"github.com/meridien-engine/meridien-engine/internal/repository"
)

type RESTHandler struct {
	erp     *erp.Service
	queries *db.Queries
}

func NewRESTHandler(erpSvc *erp.Service, queries *db.Queries) *RESTHandler {
	return &RESTHandler{erp: erpSvc, queries: queries}
}

func (h *RESTHandler) MountRoutes(r chi.Router) {
	r.Get("/orders", h.ListOrders)
	r.Get("/feed", h.LiveFeed)
	r.Get("/analytics/revenue", h.RevenueChart)
	r.Get("/analytics/overview", h.DashboardOverview)
	r.Get("/customers", h.ListCustomers)
}

func (h *RESTHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	// IMPORTANT: For local development with frontend running on a different port,
	// we need to set CORS headers.
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	ctx := r.Context()
	
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	
	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// Fetch domain orders
	orders, err := h.erp.ListOrders(ctx, int32(limit), int32(offset))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
    
    // Map to the JSON structure expected by the SolidJS DataGrid component.
    type OrderResponse struct {
        ID           string  `json:"id"`
        CustomerName string  `json:"customerName"` // Using a placeholder since we haven't implemented customer joins yet
        Date         string  `json:"date"`
        Channel      string  `json:"channel"`
        Status       string  `json:"status"`
        Amount       float64 `json:"amount"`
        AIHandled    bool    `json:"aiHandled"`
    }

    var resp []OrderResponse
    for _, o := range orders {
        resp = append(resp, OrderResponse{
            // We use the first 8 characters of UUID as a visual order ID
            ID:           "ORD-" + o.ID.String()[:8],
            CustomerName: "Customer " + o.CustomerID.String()[:4],
            Date:         o.CreatedAt.Format("2006-01-02T15:04:05Z"),
            Channel:      string(o.Source),
            Status:       string(o.Status),
            Amount:       o.TotalPrice.InexactFloat64(),
            AIHandled:    o.Source == "agent",
        })
    }

	w.Header().Set("Content-Type", "application/json")
    if resp == nil {
        resp = []OrderResponse{}
    }
	json.NewEncoder(w).Encode(resp)
}

func (h *RESTHandler) LiveFeed(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	ctx := r.Context()
	
	// Try to get businessID from context via repository helper.
	// We'll import "github.com/meridien-engine/meridien-engine/internal/repository" in the file imports
	businessIDStr, err := repository.BusinessIDFromContext(ctx)
	if err != nil {
		http.Error(w, "unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}

	businessID, err := uuid.Parse(businessIDStr)
	if err != nil {
		http.Error(w, "invalid business id: "+err.Error(), http.StatusBadRequest)
		return
	}

	feedRows, err := h.queries.GetLiveFeed(ctx, businessID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type FeedResponse struct {
		ID          string `json:"id"`
		Type        string `json:"type"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Timestamp   string `json:"timestamp"`
		Meta        string `json:"meta,omitempty"`
	}

	var resp []FeedResponse
	for _, row := range feedRows {
		itemType := "info"
		title := ""
		
		if row.ItemType == "interaction" {
			itemType = "info"
			title = "New Interaction"
			if row.TitleMeta != "" {
				title = title + " (" + row.TitleMeta + ")"
			}
		} else if row.ItemType == "order" {
			if row.TitleMeta == "completed" {
				itemType = "success"
			} else if row.TitleMeta == "pending" {
				itemType = "warning"
			} else {
				itemType = "info" // cancelled etc
			}
			title = "Order Placed"
		}

		resp = append(resp, FeedResponse{
			ID:          row.ID.String(),
			Type:        itemType,
			Title:       title,
			Description: row.Description,
			Timestamp:   row.CreatedAt.Format("2006-01-02T15:04:05Z"),
			Meta:        row.TitleMeta,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if resp == nil {
		resp = []FeedResponse{}
	}
	json.NewEncoder(w).Encode(resp)
}

func (h *RESTHandler) RevenueChart(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	ctx := r.Context()
	
	businessIDStr, err := repository.BusinessIDFromContext(ctx)
	if err != nil {
		http.Error(w, "unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}

	businessID, err := uuid.Parse(businessIDStr)
	if err != nil {
		http.Error(w, "invalid business id: "+err.Error(), http.StatusBadRequest)
		return
	}

	rows, err := h.queries.GetRevenueLast7Days(ctx, businessID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type RevenueData struct {
		Label string  `json:"label"`
		Value float64 `json:"value"`
	}

	var resp []RevenueData
	for _, row := range rows {
		resp = append(resp, RevenueData{
			Label: row.Date.Format("Mon"),
			Value: row.TotalRevenue.InexactFloat64(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if resp == nil {
		resp = []RevenueData{}
	}
	json.NewEncoder(w).Encode(resp)
}

func (h *RESTHandler) DashboardOverview(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	ctx := r.Context()
	
	businessIDStr, err := repository.BusinessIDFromContext(ctx)
	if err != nil {
		http.Error(w, "unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}

	businessID, err := uuid.Parse(businessIDStr)
	if err != nil {
		http.Error(w, "invalid business id: "+err.Error(), http.StatusBadRequest)
		return
	}

	metrics, err := h.queries.GetDashboardOverviewMetrics(ctx, businessID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type OverviewResponse struct {
		TotalRevenue     float64 `json:"total_revenue"`
		OrdersProcessed  int64   `json:"orders_processed"`
		InterceptionRate float64 `json:"interception_rate"`
		PendingReview    int64   `json:"pending_review"`
	}

	resp := OverviewResponse{
		TotalRevenue:     metrics.TotalRevenue.InexactFloat64(),
		OrdersProcessed:  metrics.OrdersProcessed,
		InterceptionRate: metrics.InterceptionRate.InexactFloat64(),
		PendingReview:    metrics.PendingReview,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *RESTHandler) ListCustomers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	ctx := r.Context()
	
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	
	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	businessIDStr, err := repository.BusinessIDFromContext(ctx)
	if err != nil {
		http.Error(w, "unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}

	businessID, err := uuid.Parse(businessIDStr)
	if err != nil {
		http.Error(w, "invalid business id: "+err.Error(), http.StatusBadRequest)
		return
	}

	arg := db.ListCustomersParams{
		Limit:      int32(limit),
		Offset:     int32(offset),
		BusinessID: businessID,
	}

	customers, err := h.queries.ListCustomers(ctx, arg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type CustomerResponse struct {
		ID             string `json:"id"`
		Name           string `json:"name"`
		Tier           string `json:"tier"`
		Summary        string `json:"summary"`
		PrimaryChannel string `json:"primary_channel"`
		JoinedAt       string `json:"joined_at"`
	}

	var resp []CustomerResponse
	for _, c := range customers {
		summary := ""
		if c.SemanticSummary.Valid {
			summary = c.SemanticSummary.String
		}
		
		name := ""
		if c.UnifiedName.Valid {
			name = c.UnifiedName.String
		}

		resp = append(resp, CustomerResponse{
			ID:             c.ID.String(),
			Name:           name,
			Tier:           c.CustomerTier,
			Summary:        summary,
			PrimaryChannel: c.PrimaryChannel,
			JoinedAt:       c.CreatedAt.Format(time.RFC3339),
		})
	}

	if resp == nil {
		resp = []CustomerResponse{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}


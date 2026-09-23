package line

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/odysight/crm/pkg/response"
)

type Handler struct {
	service       *Service
	channelSecret string
}

func NewHandler(service *Service, channelSecret string) *Handler {
	return &Handler{service: service, channelSecret: channelSecret}
}

func Routes(h *Handler) chi.Router {
	r := chi.NewRouter()
	// Public endpoint (mounted outside the auth group): authenticity comes
	// from the X-Line-Signature HMAC, not from a staff session.
	r.Post("/webhook", h.Webhook)
	return r
}

type webhookSource struct {
	Type   string `json:"type"`
	UserID string `json:"userId"`
}

type webhookEvent struct {
	Type       string        `json:"type"`
	ReplyToken string        `json:"replyToken"`
	Source     webhookSource `json:"source"`
}

type webhookBody struct {
	Events []webhookEvent `json:"events"`
}

// Webhook handles POST /api/v1/line/webhook — LINE OA deliveries.
// Always 2xx on success so LINE stops retrying; the lead upsert is
// idempotent, so LINE's own retries on failures are safe too.
func (h *Handler) Webhook(w http.ResponseWriter, r *http.Request) {
	if h.channelSecret == "" {
		response.Error(w, http.StatusServiceUnavailable, "line integration is not configured")
		return
	}
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "unreadable body")
		return
	}
	if !VerifySignature(h.channelSecret, raw, r.Header.Get("X-Line-Signature")) {
		response.Error(w, http.StatusUnauthorized, "invalid line signature")
		return
	}
	var body webhookBody
	if err := json.Unmarshal(raw, &body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	events := make([]Event, 0, len(body.Events))
	for _, e := range body.Events {
		if e.Source.Type != "user" || e.Source.UserID == "" {
			continue
		}
		events = append(events, Event{Type: e.Type, ReplyToken: e.ReplyToken, UserID: e.Source.UserID})
	}
	if _, err := h.service.HandleEvents(r.Context(), events); err != nil {
		response.Error(w, http.StatusBadGateway, "line event handling failed")
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

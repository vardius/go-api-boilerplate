package httpapi

import (
	"net/http"
	"time"

	"github.com/vardius/go-api-boilerplate/internal/auth/app"
	"github.com/vardius/go-api-boilerplate/internal/auth/store"
	"github.com/vardius/go-api-boilerplate/internal/platform/authn"
	"github.com/vardius/go-api-boilerplate/internal/platform/httpx"
)

// A client's JSON never holds its secret, nor its hash.
type clientJSON struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	RedirectURIs []string  `json:"redirect_uris"`
	Scopes       []string  `json:"scopes"`
	CreatedAt    time.Time `json:"created_at"`
}

func newClientJSON(c store.Client) clientJSON {
	return clientJSON{ID: c.ID, Name: c.Name, RedirectURIs: c.RedirectURIs, Scopes: c.Scopes, CreatedAt: c.CreatedAt}
}

type clientsResponse struct {
	Clients    []clientJSON `json:"clients"`
	Limit      int          `json:"limit"`
	NextCursor string       `json:"next_cursor,omitzero"`
}

// defaultClientsPage is how many clients a page holds unless ?limit= says.
const defaultClientsPage = 20

func (h *handler) clients(w http.ResponseWriter, r *http.Request) error {
	id, _ := authn.FromContext(r.Context())
	page, err := httpx.ParsePage(r, defaultClientsPage, app.MaxClientsPage)
	if err != nil {
		return err
	}
	list, err := h.app.Clients(r.Context(), id.UserID, page.Limit, page.Cursor)
	if err != nil {
		return err
	}
	out := clientsResponse{Clients: make([]clientJSON, len(list.Clients)), Limit: page.Limit, NextCursor: list.Next}
	for i, c := range list.Clients {
		out.Clients[i] = newClientJSON(c)
	}
	return httpx.WriteJSON(w, http.StatusOK, out)
}

type createClientRequest struct {
	Name         string   `json:"name"`
	RedirectURIs []string `json:"redirect_uris"`
	Scopes       []string `json:"scopes"`
}

type createClientResponse struct {
	Client       clientJSON `json:"client"`
	ClientSecret string     `json:"client_secret"`
}

func (h *handler) createClient(w http.ResponseWriter, r *http.Request) error {
	id, _ := authn.FromContext(r.Context())
	in, err := httpx.DecodeJSON[createClientRequest](r, maxBody)
	if err != nil {
		return err
	}
	c, secret, err := h.app.CreateClient(r.Context(), id.UserID, app.NewClient(in))
	if err != nil {
		return err
	}
	return httpx.WriteJSON(w, http.StatusCreated, createClientResponse{Client: newClientJSON(c), ClientSecret: secret.Reveal()})
}

func (h *handler) deleteClient(w http.ResponseWriter, r *http.Request) error {
	id, _ := authn.FromContext(r.Context())
	if err := h.app.DeleteClient(r.Context(), id.UserID, r.PathValue("id")); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

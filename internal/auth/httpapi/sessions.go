package httpapi

import (
	"errors"
	"net/http"
	"time"
	"uuid"

	"github.com/vardius/go-api-boilerplate/internal/auth/app"
	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
	"github.com/vardius/go-api-boilerplate/internal/platform/authn"
	"github.com/vardius/go-api-boilerplate/internal/platform/httpx"
)

type sessionJSON struct {
	ID        uuid.UUID `json:"id"`
	ExpiresAt time.Time `json:"expires_at"`
}

type sessionResponse struct {
	Session sessionJSON `json:"session"`
}

func newSessionResponse(t app.Tokens) sessionResponse {
	return sessionResponse{Session: sessionJSON{ID: t.Grant.ID, ExpiresAt: t.Grant.ExpiresAt}}
}

type signInRequest struct {
	Code string `json:"code"`
}

func (h *handler) signIn(w http.ResponseWriter, r *http.Request) error {
	in, err := httpx.DecodeJSON[signInRequest](r, maxBody)
	if err != nil {
		return err
	}
	t, err := h.app.SignIn(r.Context(), in.Code, r.UserAgent())
	if err != nil {
		return err
	}
	h.setSession(w, t)
	return httpx.WriteJSON(w, http.StatusCreated, newSessionResponse(t))
}

func (h *handler) refresh(w http.ResponseWriter, r *http.Request) error {
	t, err := h.app.RefreshSession(r.Context(), refreshToken(r))
	if errors.Is(err, apperr.ErrUnauthorized) {
		h.clearSession(w)
	}
	if err != nil {
		return err
	}
	h.setSession(w, t)
	return httpx.WriteJSON(w, http.StatusOK, newSessionResponse(t))
}

func (h *handler) signOut(w http.ResponseWriter, r *http.Request) error {
	if err := h.app.SignOut(r.Context(), refreshToken(r)); err != nil {
		return err
	}
	h.clearSession(w)
	w.WriteHeader(http.StatusNoContent)
	return nil
}

type sessionListJSON struct {
	ID         uuid.UUID `json:"id"`
	CreatedAt  time.Time `json:"created_at"`
	LastUsedAt time.Time `json:"last_used_at"`
	UserAgent  string    `json:"user_agent"`
	Current    bool      `json:"current"`
}

type sessionsResponse struct {
	Sessions []sessionListJSON `json:"sessions"`
}

func (h *handler) sessions(w http.ResponseWriter, r *http.Request) error {
	id, _ := authn.FromContext(r.Context())
	list, err := h.app.Sessions(r.Context(), id.UserID, refreshToken(r))
	if err != nil {
		return err
	}
	out := sessionsResponse{Sessions: make([]sessionListJSON, len(list))}
	for i, s := range list {
		out.Sessions[i] = sessionListJSON{ID: s.ID, CreatedAt: s.CreatedAt, LastUsedAt: s.LastUsedAt, UserAgent: s.UserAgent, Current: s.Current}
	}
	return httpx.WriteJSON(w, http.StatusOK, out)
}

var errNoSession = apperr.New(apperr.ErrNotFound, "No such session.")

func (h *handler) revokeSession(w http.ResponseWriter, r *http.Request) error {
	id, _ := authn.FromContext(r.Context())
	session, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		return errNoSession
	}
	if err := h.app.RevokeSession(r.Context(), id.UserID, session); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

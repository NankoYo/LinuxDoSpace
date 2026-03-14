package httpapi

import (
	"net/http"

	"linuxdospace/backend/internal/service"
)

// handleMyPOWStatus returns the authenticated user's current proof-of-work
// dashboard state, including the active challenge and daily claim counters.
func (a *API) handleMyPOWStatus(w http.ResponseWriter, r *http.Request) {
	_, user, ok := a.requireActor(w, r)
	if !ok {
		return
	}

	item, err := a.powService.GetMyStatus(r.Context(), *user)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// handleCreateMyPOWChallenge replaces any older active challenge with one new
// puzzle for the authenticated user.
func (a *API) handleCreateMyPOWChallenge(w http.ResponseWriter, r *http.Request) {
	session, user, ok := a.requireActor(w, r)
	if !ok {
		return
	}
	if !a.enforceCSRF(w, r, session) {
		return
	}

	var request service.GeneratePOWChallengeRequest
	if err := decodeJSONBody(r, &request); err != nil {
		writeError(w, err)
		return
	}

	item, err := a.powService.CreateChallenge(r.Context(), *user, request)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

// handleClaimMyPOWChallenge verifies one submitted nonce and grants the reward
// when the backend confirms that the active challenge is solved.
func (a *API) handleClaimMyPOWChallenge(w http.ResponseWriter, r *http.Request) {
	session, user, ok := a.requireActor(w, r)
	if !ok {
		return
	}
	if !a.enforceCSRF(w, r, session) {
		return
	}

	var request service.SubmitPOWChallengeRequest
	if err := decodeJSONBody(r, &request); err != nil {
		writeError(w, err)
		return
	}

	item, err := a.powService.SubmitChallenge(r.Context(), *user, request)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

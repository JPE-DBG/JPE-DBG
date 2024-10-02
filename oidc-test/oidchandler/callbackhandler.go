package oidchandler

import "net/http"

// HandleCallback handles the OAuth2 callback from the OIDC provider.
// It exchanges the authorization code for an OAuth2 token, retrieves the user's groups,
// saves the token and groups to the session, clears the state cookie, and redirects to the menu page.
func (h *OidcHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	token, err := h.exchangeToken(r.Context(), r)
	if err != nil {
		http.Error(w, "failed to exchange token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	groups, err := h.groups(r.Context(), token)
	if err != nil {
		http.Error(w, "failed to get groups: "+err.Error(), http.StatusInternalServerError)
		return
	}

	hasReadAccess := isGroupMember(groups, readerGroup)

	values := map[string]any{
		sessionGroupKey: groups,
		sessionTokenKey: *token,
		loginIndicator:  true,
		readAccess:      hasReadAccess,
	}
	h.saveToSession(r.Context(), values)

	// clear the state cookie, it's not needed anymore
	h.clearStateCookie(w)

	http.Redirect(w, r, "/menu", http.StatusSeeOther)
}

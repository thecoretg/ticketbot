package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

const cookieName = "tb_session"

func convertID(r *http.Request) (int, error) {
	return strconv.Atoi(r.PathValue("id"))
}

// decodeJSON reads the request body into v. Unknown fields are allowed; handlers that must reject
// them build their own decoder with DisallowUnknownFields.
func decodeJSON(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// setSessionCookie writes the session cookie. A negative maxAge clears it.
func setSessionCookie(w http.ResponseWriter, token string, maxAge time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(maxAge / time.Second),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

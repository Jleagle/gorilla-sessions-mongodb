package gsm

import (
	"net/http"

	"github.com/gorilla/sessions"
)

type TokenGetSeter interface {
	GetToken(r *http.Request, name string) (string, error)
	SetToken(w http.ResponseWriter, name, value string, options *sessions.Options)
}

type CookieToken struct{}

func (c *CookieToken) GetToken(req *http.Request, name string) (string, error) {

	cookie, err := req.Cookie(name)
	if err != nil {
		return "", err
	}

	return cookie.Value, nil
}

// SetToken sets the sessions cookie
func (c *CookieToken) SetToken(w http.ResponseWriter, name, value string, options *sessions.Options) {
	http.SetCookie(w, sessions.NewCookie(name, value, options))
}

package settings

import (
	"net/http"
	"os"

	"github.com/gorilla/sessions"
	"github.com/pkg/errors"
)

var CookieSession = "isitsql-session"

func GetSession(r *http.Request) (*sessions.Session, error) {
	sessionKey, err := DecryptString(os.Getenv("ISITSQL_SESSION_KEY"))
	if err != nil {
		return &sessions.Session{}, errors.Wrap(err, "settings.decryptstring")
	}

	var store = sessions.NewCookieStore(sessionKey)
	session, err := store.Get(r, CookieSession)
	if err != nil {
		return &sessions.Session{}, errors.Wrap(err, "store.get")
	}
	return session, nil
}

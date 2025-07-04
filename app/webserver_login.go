package app

import (
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/scalesql/isitsql/internal/ad"
	"github.com/scalesql/isitsql/internal/settings"
)

func loginPage(w http.ResponseWriter, r *http.Request) {

	var err error

	//m := make(map[string]string)
	context := struct {
		Context
		//Values   map[string]string
		Message  string
		Error    error
		User     ad.User
		LoggedIn bool
	}{
		Context: Context{
			Title:       "Login",
			UnixNow:     time.Now().Unix() * 1000,
			ErrorList:   getServerErrorList(),
			HeaderRight: fmt.Sprintf("Refreshed: %s (%s)", time.Now().Format("15:04:05"), version),
			TagList:     globalTagList.getTags(),
			AppConfig:   getGlobalConfig(),
		},
		//Values:  m,
		Message: "",
	}

	// sessionKey, err := settings.DecryptString(os.Getenv("ISITSQL_SESSION_KEY"))
	// if err != nil {
	// 	context.Error = errors.Wrap(err, "settings.decryptstring")
	// 	GLOBAL_RINGLOG.Enqueue(errors.Wrap(err, "settings.decryptstring").Error())
	// 	renderFSDynamic(w, "login-get", context)
	// 	return
	// }

	// var store = sessions.NewCookieStore(sessionKey)
	// session, err := store.Get(r, globalCookieSession)
	// if err != nil {
	// 	context.Error = errors.Wrap(err, "store.get")
	// 	GLOBAL_RINGLOG.Enqueue(errors.Wrap(err, "store.get").Error())
	// 	renderFSDynamic(w, "login-get", context)
	// 	return
	// }

	session, err := settings.GetSession(r)
	if err != nil {
		context.Error = errors.Wrap(err, "getsession")
		GLOBAL_RINGLOG.Enqueue(errors.Wrap(err, "getsession").Error())
		renderFSDynamic(w, "login-get", context)
		return
	}

	var u ad.User
	if !session.IsNew {
		context.LoggedIn = true
		u.Account = fmt.Sprintf("%v", session.Values["account"])
		u.Name = fmt.Sprintf("%v", session.Values["name"])
		u.Admin, _ = session.Values["admin"].(bool)
	} else {
		u = ad.User{}
	}
	context.User = u

	// Process a POST
	if r.Method == "POST" {

		// err = testLogin2("gauss@ldap.forumsys.com", "password", true)
		// if err != nil {
		// 	fmt.Println(err)
		// }

		err = r.ParseForm()
		if err != nil {
			context.Error = errors.Wrap(err, "error parsing form")
			GLOBAL_RINGLOG.Enqueue(errors.Wrap(err, "error parsing form").Error())
			renderFSDynamic(w, "login-get", context)
			return
		}

		button := r.FormValue("submit")
		if button == "logout" {
			session.Options.MaxAge = -1
			err = session.Save(r, w)
			if err != nil {
				context.Error = errors.Wrap(err, "session.save")
				context.Message = ""
				GLOBAL_RINGLOG.Enqueue(errors.Wrap(err, "session.save").Error())
				renderFSDynamic(w, "login-get", context)
				return
			}

			context.LoggedIn = false
			context.User = ad.User{}
			context.Message = "Logged out"
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		// Split the domain part out to use on the groups
		_, _, err = ad.ParseName(r.FormValue("userName"))
		if err != nil && !DEV {
			context.Error = errors.Wrap(err, "ad.parsename")
			renderFSDynamic(w, "login-get", context)
			return
		}

		// err = ad.Login(r.FormValue("userName"), r.FormValue("password"), true)
		// if err != nil {
		// 	context.Error = errors.Wrap(err, "ad.login")
		// 	renderFSDynamic(w, "login-get", context)
		// 	return
		// }

		stgs, err := settings.ReadConfig()
		if err != nil {
			context.Error = errors.Wrap(err, "settings.readconfig")
			GLOBAL_RINGLOG.Enqueue(errors.Wrap(err, "settings.readconfig").Error())
			renderFSDynamic(w, "login-get", context)
			return
		}
		group := stgs.AdminDomainGroup
		user, err := ad.Validate(r.FormValue("userName"), r.FormValue("password"), group)
		if err != nil {
			context.Error = errors.Wrap(err, "ad.userhasgroup")
			renderFSDynamic(w, "login-get", context)
			return
		}

		if user.Admin {
			session.Values["account"] = user.Account
			session.Values["name"] = user.Name
			session.Values["admin"] = user.Admin
			err = session.Save(r, w)
			if err != nil {
				context.Error = errors.Wrap(err, "session.save")
				renderFSDynamic(w, "login-get", context)
				return
			}
			context.Message = fmt.Sprintf("User is in group: %s", group)
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		} else {
			//lint:ignore ST1005 error is dispalyed to user
			context.Error = fmt.Errorf("User is NOT in group: %s", group)
		}
	}

	renderFSDynamic(w, "login-get", context)
}

func logoutPage(w http.ResponseWriter, r *http.Request) {
	session, err := settings.GetSession(r)
	if err != nil {
		GLOBAL_RINGLOG.Enqueue(errors.Wrap(err, "logout: getsession").Error())
		http.Redirect(w, r, "/login", http.StatusFound)
	}

	if !session.IsNew {
		session.Options.MaxAge = -1
		err := session.Save(r, w)
		if err != nil {
			GLOBAL_RINGLOG.Enqueue(errors.Wrap(err, "logout: session.save").Error())
		}
	}
	http.Redirect(w, r, "/login", http.StatusFound)
}

// func getSession(r *http.Request) (*sessions.Session, error) {
// 	sessionKey, err := settings.DecryptString(os.Getenv("ISITSQL_SESSION_KEY"))
// 	if err != nil {
// 		return &sessions.Session{}, errors.Wrap(err, "settings.decryptstring")
// 	}

// 	var store = sessions.NewCookieStore(sessionKey)
// 	session, err := store.Get(r, globalCookieSession)
// 	if err != nil {
// 		return &sessions.Session{}, errors.Wrap(err, "store.get")
// 	}
// 	return session, nil
// }

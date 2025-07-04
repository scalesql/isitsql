// Package ldap provides a simple ldap client to authenticate,
// retrieve basic information and groups for a user.
package app

import (
	"crypto/tls"
	"errors"
	"fmt"

	"github.com/go-ldap/ldap/v3"
)

type LDAPClient struct {
	Conn         *ldap.Conn
	Host         string
	Port         int
	UseSSL       bool
	BindDN       string
	BindPassword string
	GroupFilter  string // e.g. "(memberUid=%s)"
	UserFilter   string // e.g. "(uid=%s)"
	Base         string
	Attributes   []string
}

// Connect connects to the ldap backend
func (lc *LDAPClient) Connect() error {
	if lc.Conn == nil {
		var l *ldap.Conn
		var err error
		address := fmt.Sprintf("%s:%d", lc.Host, lc.Port)
		if !lc.UseSSL {
			l, err = ldap.Dial("tcp", address)
			if err != nil {
				return err
			}

			// Reconnect with TLS
			/* #nosec G402 required since no server name */
			err = l.StartTLS(&tls.Config{
				InsecureSkipVerify:       true,
				MinVersion:               tls.VersionTLS12,
				PreferServerCipherSuites: true,
			})
			if err != nil {
				return err
			}
		} else {
			/* #nosec G402 required since no server name */
			l, err = ldap.DialTLS("tcp", address, &tls.Config{
				InsecureSkipVerify:       true,
				MinVersion:               tls.VersionTLS12,
				PreferServerCipherSuites: true})
			if err != nil {
				return err
			}
		}

		lc.Conn = l
	}
	return nil
}

// Close closes the ldap backend connection
func (lc *LDAPClient) Close() {
	if lc.Conn != nil {
		lc.Conn.Close()
		lc.Conn = nil
	}
}

// Authenticate authenticates the user against the ldap backend
func (lc *LDAPClient) Authenticate(username, password string) (bool, map[string]string, error) {
	err := lc.Connect()
	if err != nil {
		return false, nil, err
	}

	// First bind with a read only user
	if lc.BindDN != "" && lc.BindPassword != "" {
		err := lc.Conn.Bind(lc.BindDN, lc.BindPassword)
		if err != nil {
			return false, nil, err
		}
	}

	attributes := append(lc.Attributes, "dn")
	// Search for the given username
	searchRequest := ldap.NewSearchRequest(
		lc.Base,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf(lc.UserFilter, username),
		attributes,
		nil,
	)

	sr, err := lc.Conn.Search(searchRequest)
	if err != nil {
		return false, nil, err
	}

	if len(sr.Entries) < 1 {
		return false, nil, errors.New("user does not exist")
	}

	if len(sr.Entries) > 1 {
		return false, nil, errors.New("too many entries returned")
	}

	userDN := sr.Entries[0].DN
	user := map[string]string{}
	for _, attr := range lc.Attributes {
		user[attr] = sr.Entries[0].GetAttributeValue(attr)
	}

	// Bind as the user to verify their password
	err = lc.Conn.Bind(userDN, password)
	if err != nil {
		return false, user, err
	}

	// Rebind as the read only user for any further queries
	if lc.BindDN != "" && lc.BindPassword != "" {
		err = lc.Conn.Bind(lc.BindDN, lc.BindPassword)
		if err != nil {
			return true, user, err
		}
	}

	return true, user, nil
}

// GetGroupsOfUser returns the group for a user
func (lc *LDAPClient) GetGroupsOfUser(username string) ([]string, error) {
	err := lc.Connect()
	if err != nil {
		return nil, err
	}

	searchRequest := ldap.NewSearchRequest(
		lc.Base,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf(lc.GroupFilter, username),
		[]string{"cn"}, // can it be something else than "cn"?
		nil,
	)
	sr, err := lc.Conn.Search(searchRequest)
	if err != nil {
		return nil, err
	}
	groups := []string{}
	for _, entry := range sr.Entries {
		groups = append(groups, entry.GetAttributeValue("cn"))
	}
	return groups, nil
}

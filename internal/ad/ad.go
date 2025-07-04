package ad

import (
	"crypto/tls"
	"fmt"
	"strings"

	"github.com/go-ldap/ldap/v3"
	"github.com/pkg/errors"
)

type User struct {
	Account string
	Name    string
	Admin   bool
}

// ParseName returns user and domain from user@domain
func ParseName(userName string) (string, string, error) {
	parts := strings.Split(userName, "@")
	if len(parts) != 2 {
		return "", "", errors.New("expected username@domain format")
	}
	return parts[0], parts[1], nil
}

// Login atttempts to login
func Login(user, password string, verbose bool) error {
	_, domain, err := ParseName(user)
	if err != nil {
		return errors.Wrap(err, "parsename")
	}

	conn, err := getLdapConnection(domain, 389)
	if err != nil {
		return errors.Wrap(err, "getldapconnection")
	}
	defer conn.Close()

	//err = l.Bind("gauss@mathematicians.example.com", "password")
	//err = l.Bind("cn=gauss,dc=example,dc=com", "password")
	err = conn.Bind(user, password)
	if err != nil {
		if strings.HasPrefix(err.Error(), "LDAP Result Code 49") {
			return errors.New("invalid credentials")
		}
		// fmt.Printf("%#v\n", []byte(err.Error()))
		// b = bytes.Trim(b, "\x00")
		return errors.Wrap(err, "conn.bind")
	}

	return nil
}

// // Determines whether a user is a member of a group
// func UserHasGroup(user, password, group string) (bool, error) {
// 	groups, err := Groups(user, password)
// 	if err != nil {
// 		return false, errors.Wrap(err, "groups")
// 	}
// 	for _, g := range groups {
// 		if strings.EqualFold(group, g) {
// 			return true, nil
// 		}
// 	}
// 	return false, nil
// }

// Groups returns the groups for a user
func Validate(user, password, group string) (User, error) {
	u := User{Account: user}
	_, domain, err := ParseName(user)
	if err != nil {
		return u, errors.Wrap(err, "parsename")
	}

	conn, err := getLdapConnection(domain, 389)
	if err != nil {
		return u, errors.Wrap(err, "getddapconnection")
	}
	defer conn.Close()

	err = conn.Bind(user, password)
	if err != nil {
		if strings.HasPrefix(err.Error(), "LDAP Result Code 49") {
			return u, errors.New("invalid credentials")
		}
		return u, errors.Wrap(err, "bind")
	}

	// searchRequest := ldap.NewSearchRequest(
	// 	"dc=infrastructure,dc=us,dc=loc",
	// 	ldap.ScopeWholeSubtree,
	// 	ldap.NeverDerefAliases,
	// 	0,
	// 	0,
	// 	false,
	// 	//"(samAccountName=test)", // This works
	// 	"(userPrincipalName=bgraziano@domain.us.loc)", // This works
	// 	//[]string{"cn", "name", "memberOf", "displayName"}, // can it be something else than "cn"?
	// 	// []string{"*"}, // this gets all
	// 	[]string{"memberOf"},
	// 	nil,
	// )
	dc := domainToDC(domain)
	userPrincipal := fmt.Sprintf("(userPrincipalName=%s)", user)

	//fmt.Println("dc: ", dc)
	//mt.Println("userPrincipal: ", userPrincipal)

	sr := ldap.NewSearchRequest(
		dc,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,
		0,
		false,
		//"(samAccountName=test)", // This works
		//"(userPrincipalName=test@adtest.local)", // This works
		userPrincipal,
		//[]string{"cn", "name", "memberOf", "displayName"}, // can it be something else than "cn"?
		// []string{"*"}, // this gets all
		[]string{"memberOf"},
		nil,
	)

	results, err := conn.Search(sr)
	if err != nil {
		return u, errors.Wrap(err, "search")
	}

	if len(results.Entries) > 1 {
		bad := make([]string, 0)
		for _, e := range results.Entries {
			bad = append(bad, fmt.Sprintf("%v", e))
		}
		return u, fmt.Errorf("multiple ldap entries: %s", strings.Join(bad, ", "))
	}

	entry := results.Entries[0]
	name := parseCN(entry.DN)
	if name == "" {
		return u, errors.Wrap(err, "parsecn")
	}
	u.Name = name

	groups := results.Entries[0].GetAttributeValues("memberOf")
	for _, g := range groups {
		groupName := parseCN(g)
		if strings.EqualFold(group, groupName) {
			u.Admin = true
		}
	}

	return u, nil
}

func getLdapConnection(domain string, _ int) (*ldap.Conn, error) {
	var conn *ldap.Conn
	conn, err := ldap.Dial("tcp", fmt.Sprintf("%s:%d", domain, 389))
	if err != nil {
		return nil, errors.Wrap(err, "ldap.dial")
	}

	// Reconnect with TLS
	/* #nosec G402 required since no server name */
	err = conn.StartTLS(&tls.Config{
		InsecureSkipVerify:       true,
		MinVersion:               tls.VersionTLS12,
		PreferServerCipherSuites: true,
	})
	if err != nil {
		return nil, errors.Wrap(err, "conn.starttls")
	}

	return conn, nil
}

func domainToDC(domain string) string {
	parts := strings.Split(domain, ".")

	var dcparts []string

	for _, k := range parts {
		dc := fmt.Sprintf("dc=%s", k)
		dcparts = append(dcparts, dc)
	}
	return strings.Join(dcparts, ",")
}

func parseCN(s string) string {
	if s == "" {
		return ""
	}
	//CN=eus-sql-dpa-sysadmin,OU=org-sqlgroups,OU=org,DC=core,DC=org,DC=us,DC=loc
	csvParts := strings.Split(s, ",")
	if len(csvParts) == 0 {
		return ""
	}

	first := csvParts[0]
	cn := strings.Split(first, "=")
	if len(cn) != 2 {
		return ""
	}
	if cn[0] != "CN" {
		return ""
	}
	return cn[1]
}

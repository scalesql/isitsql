package settings

import (
	"fmt"
	"testing"
)

func TestCredentials(t *testing.T) {
	var err error
	err = SetupConfigDir()
	if err != nil {
		t.Error("Can't setup config dir: ", err)
	}

	k, err := AddSQLCredential("test", "myLogin", "myLogin")
	if err != nil {
		t.Error("AddSQLCredential: ", err)
	}
	fmt.Println("CredentialKey: ", k)

	c2, err := GetSQLCredential(k)
	if err != nil {
		t.Error("getSQLCredential: ", err)
	}

	if c2.Password != "myLogin" {
		t.Error("Passwords don't match", "myLogin", c2.Password)
	}

	c2.Name = "NewName"
	err = SaveSQLCredential(*c2)
	if err != nil {
		t.Error("Saving credential: ", err)
	}
}

func TestMissingCredential(t *testing.T) {
	var err error
	err = SetupConfigDir()
	if err != nil {
		t.Error("Can't setup config dir: ", err)
	}

	_, err = GetSQLCredential("bang")
	if err != ErrNotFound {
		t.Error("getSQLCredential-missing: ", err)
	}
}

func TestListSQLCredentails(t *testing.T) {
	r, err := ListSQLCredentials()
	if err != nil {
		t.Error("ListSQLCredentials", err)
	}

	if len(r) == 0 {
		t.Error("no credentials found")
	}
}

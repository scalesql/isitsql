package settings

import (
	"log"
	"testing"
)

func TestSQLSettings(t *testing.T) {
	var err error
	err = SetupConfigDir()
	if err != nil {
		t.Error("Can't setup config dir: ", err)
	}

	_, err = ReadConnections()
	if err != nil {
		t.Error("ReadConnections: ", err)
	}

	s1 := SQLServer{
		FQDN:              "xxyyzz",
		TrustedConnection: true,
	}

	_, err = AddSQLServer(s1)
	if err != nil {
		t.Error("AddSQLServer: ", err)
	}

	// err = printJSON()
	// if err != nil {
	// 	t.Error("printJSON: ", err)
	// }

	_, err = ReadConnections()
	if err != nil {
		t.Error("ReadConnections: ", err)
	}

	s3 := SQLServer{
		FQDN:              "boot",
		FriendlyName:      "mikey",
		TrustedConnection: false,
		Login:             "myLogin",
		Password:          "encrypted",
		Tags:              []string{"a", "b", "c"},
	}

	key, err := AddSQLServer(s3)
	if err != nil {
		t.Error("AddSQLServer-3: ", err)
	}
	log.Println("server key: ", key)

	// err = printJSON()
	// if err != nil {
	// 	t.Error("printJSON: ", err)
	// }

}

func TestSaving(t *testing.T) {
	log.Println("Test Saving...")
	var err error
	err = SetupConfigDir()
	if err != nil {
		t.Error("Can't setup config dir: ", err)
	}

	s3 := SQLServer{
		FQDN:              "saveTest",
		FriendlyName:      "mikey",
		TrustedConnection: false,
		Login:             "myLogin",
		Password:          "zoot@",
		Tags:              []string{"a", "b", "c"},
	}

	key, err := AddSQLServer(s3)
	if err != nil {
		t.Error("AddSQLServer-3: ", err)
	}
	log.Println("Added Key: ", key)

	x, err := GetSQLServer(key)
	if err != nil {
		t.Error("failed get: ", key)
	}

	x.FQDN = "New FQDN"
	x.Login = "Zip"
	x.Password = "theOriginalValue1@"

	err = SaveSQLServer(key, x)
	if err != nil {
		t.Error("save", err)
	}

	// err = printJSON()
	// if err != nil {
	// 	t.Error("printJSON: ", err)
	// }
}

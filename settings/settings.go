package settings

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/kardianos/osext"
	"github.com/pkg/errors"
)

// ErrNotFound is for settings that aren't found
var ErrNotFound = errors.New("Not Found")

// ErrDuplicate is for duplicate settings
var ErrDuplicate = errors.New("Duplicate")

// SetupConfigDir setups up the configuration directory
func SetupConfigDir() error {
	err := MakeDir("config")
	if err != nil {
		return errors.Wrap(err, "makedir")
	}
	return nil
}

// MakeDir makes a directory in the directory of the executable
func MakeDir(name string) error {
	wd, err := osext.ExecutableFolder()
	if err != nil {
		return errors.Wrap(err, "osext.executablefolder")
	}
	dir := filepath.Join(wd, name)
	_, err = os.Stat(dir)
	if err == nil { // directory exists
		return nil
	}
	err = os.Mkdir(dir, 0640)
	if err != nil {
		return errors.Wrap(err, "os.mkdir")
	}
	return nil
}

// CanSave checks with this IP address can save
func CanSave(r *http.Request) (bool, error) {
	s, err := ReadConfig()
	if err != nil {
		return false, errors.Wrap(err, "readconfig")
	}

	// If policy is open, we're done
	if s.SecurityPolicy == OpenPolicy {
		return true, nil
	}

	// Get the IP address
	ip, err := IPFromRequest(r)
	if err != nil {
		return false, errors.Wrap(err, "ipfromrequest")
	}
	localips, err := GetLocalIPs()
	if err != nil {
		return false, errors.Wrap(err, "GetLocalIPs")
	}

	for _, localip := range localips {
		if localip == ip {
			return true, nil
		}
	}

	// check if we are logged in
	session, err := GetSession(r)
	if err != nil {
		return false, errors.Wrap(err, "getsession")
	}

	admin, _ := session.Values["admin"].(bool)
	return admin, nil
}

// IPFromRequest gets the orginial IP address from an HTTP request
func IPFromRequest(req *http.Request) (string, error) {
	var err error
	ip, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return "", fmt.Errorf("userip: %q is not IP:port", req.RemoteAddr)
	}

	userIP := net.ParseIP(ip)
	if userIP == nil {
		return "", fmt.Errorf("userip: %q is not IP:port", req.RemoteAddr)
	}

	// If this isn't localhost, we're done
	if userIP.String() != "127.0.0.1" && userIP.String() != "::1" {
		return userIP.String(), nil
	}

	// if we are on localhost, let's see if we have a proxy
	for _, h := range []string{"X-Forwarded-For", "X-Real-Ip"} {
		addresses := strings.Split(req.Header.Get(h), ",")
		for i := len(addresses) - 1; i >= 0; i-- {
			ip := strings.TrimSpace(addresses[i])

			realIP, err := parseIP(ip)
			// I'm just skipping things with an error
			if realIP == "" || err != nil {
				continue
			}

			// if we find a non-local IP address we're done
			if realIP != "127.0.0.1" && realIP != "::1" {
				return realIP, nil
			}
		}
	}

	// There wasn't a better IP, so return localhost
	return userIP.String(), nil
}

func parseIP(s string) (string, error) {
	ip, _, err := net.SplitHostPort(s)
	if err == nil {
		return ip, nil
	}

	ip2 := net.ParseIP(s)
	if ip2 == nil {
		return "", errors.New("invalid IP")
	}

	return ip2.String(), nil
}

// GetLocalIPs hopefully returns all the local IP addresses
func GetLocalIPs() ([]string, error) {
	var ips []string
	var err error

	ifaces, err := net.Interfaces()
	if err != nil {
		return ips, errors.Wrap(err, "net.Interfaces")
	}
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			return ips, errors.Wrap(err, "i.Addrs")
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			// process IP address
			ips = append(ips, ip.String())
		}
	}
	return ips, nil
}

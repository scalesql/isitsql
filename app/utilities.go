package app

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kardianos/osext"
)

//lint:ignore U1000 used by templates
func durationToMediumString(a, b time.Time) string {
	if a.After(b) {
		return "invalid"
		// fmt.Println(a)
		// fmt.Println(b)
	}

	up := b.Sub(a)

	// Days
	if up.Hours() > 72 {
		h := int(up.Hours())
		d := h / 24
		s := fmt.Sprintf("%dd %dh", d, h-d*24)
		return s
	}

	if up.Minutes() > 99 {
		m := int(up.Minutes())
		h := m / 60
		s := fmt.Sprintf("%dh %dm", h, m-h*60)
		return s
	}

	if up.Seconds() > 90 {
		s := int(up.Seconds())
		m := s / 60
		return fmt.Sprintf("%dm %ds", m, s-m*60)
	}

	return fmt.Sprintf("%ds", int(up.Seconds()))
}

func durationToShortString(a, b time.Time) string {
	if a.IsZero() {
		return "never"
	}

	if a.After(b) {
		z := a.Sub(b)
		if z.Minutes() < 5 {
			return "now"
		}
		return "invalid"
		// fmt.Println(a)
		// fmt.Println(b)
	}

	up := b.Sub(a)

	// Days
	if up.Hours() > 72 {
		h := int(up.Hours())
		d := h / 24
		s := fmt.Sprintf("%dd", d)
		return s
	}

	if up.Minutes() > 99 {
		m := int(up.Minutes())
		h := m / 60
		s := fmt.Sprintf("%dh", h)
		return s
	}

	if up.Seconds() > 99 {
		s := int(up.Seconds())
		m := s / 60
		return fmt.Sprintf("%dm", m)
	}

	if up.Seconds() < 1 {
		return "now"
	}

	return fmt.Sprintf("%ds", int(up.Seconds()))
}

//lint:ignore U1000 used by templates
func getJSURL(file, fallback string) (string, error) {

	dir, err := osext.ExecutableFolder()
	if err != nil {
		return "", err
	}
	// WinLogln("Current Directory: " + dir)
	dir = dir + "/static/js"
	fullfile := filepath.Join(dir, file)
	_, err = os.Stat(fullfile)
	if os.IsNotExist(err) {
		return fallback, nil
	}

	if err != nil {
		return "", err
	}

	return "/static/js/" + file, nil
}

//lint:ignore U1000 used by templates
func arrayContainsString(a []string, s string) bool {
	for _, v := range a {
		if v == s {
			return true
		}
	}
	return false
}

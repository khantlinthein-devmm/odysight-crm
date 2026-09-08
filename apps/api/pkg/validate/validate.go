package validate

import (
	"net/mail"
	"regexp"
	"strings"
)

var phoneRe = regexp.MustCompile(`^[+\d][\d\s\-().]{5,20}$`)

func Email(v string) bool {
	v = strings.TrimSpace(v)
	if v == "" || len(v) > 254 || !strings.Contains(v, "@") {
		return false
	}
	_, err := mail.ParseAddress(v)
	return err == nil
}

func Phone(v string) bool {
	v = strings.TrimSpace(v)
	if v == "" || len(v) > 30 {
		return false
	}
	return phoneRe.MatchString(v)
}

func Length(v string, min, max int) bool {
	n := len(strings.TrimSpace(v))
	return n >= min && n <= max
}

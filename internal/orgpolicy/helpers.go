package orgpolicy

import (
	"net/url"
	"strconv"
)

func itoa(value int) string {
	return strconv.Itoa(value)
}

func parseURL(raw string) (*url.URL, error) {
	return url.Parse(raw)
}

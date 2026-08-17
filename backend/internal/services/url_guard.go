package services

import (
	"errors"
	"net"
	"net/url"
	"strings"
)

var ErrURLInvalid = errors.New("invalid url")
var ErrURLBlocked = errors.New("blocked url")

// ValidateExternalURL rejects hosts that can route a crawler to local or private services.
func ValidateExternalURL(raw string) (*url.URL, error) {
	parsed, err := url.ParseRequestURI(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Hostname() == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, ErrURLInvalid
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".internal") {
		return nil, ErrURLBlocked
	}
	if ip := net.ParseIP(host); ip != nil && isPrivateIP(ip) {
		return nil, ErrURLBlocked
	}
	return parsed, nil
}

func isPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	if ipv4 := ip.To4(); ipv4 != nil {
		return ipv4[0] == 0 || ipv4[0] == 127 || (ipv4[0] == 169 && ipv4[1] == 254)
	}
	return false
}

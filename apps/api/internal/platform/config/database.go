package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// LoadDatabaseURL is shared by the API and migration binary. Split settings
// take precedence so an old DATABASE_URL cannot override a ConfigMap change.
func LoadDatabaseURL() (string, error) {
	keys := []string{"DATABASE_HOST", "DATABASE_PORT", "DATABASE_USERNAME", "DATABASE_PASSWORD", "DATABASE_NAME", "DATABASE_ENCRYPT"}
	split := false
	for _, key := range keys {
		if os.Getenv(key) != "" {
			split = true
			break
		}
	}
	if !split {
		raw := os.Getenv("DATABASE_URL")
		if raw == "" {
			return "", errors.New("DATABASE_HOST, DATABASE_USERNAME, DATABASE_PASSWORD and DATABASE_NAME are required (or set DATABASE_URL)")
		}
		parsed, err := url.Parse(raw)
		if err != nil || parsed.Scheme != "sqlserver" || parsed.Hostname() == "" || parsed.Fragment != "" {
			return "", errors.New("DATABASE_URL must be a valid sqlserver URL")
		}
		return raw, nil
	}

	for _, key := range []string{"DATABASE_HOST", "DATABASE_USERNAME", "DATABASE_PASSWORD", "DATABASE_NAME"} {
		if strings.TrimSpace(os.Getenv(key)) == "" {
			return "", fmt.Errorf("%s is required for split database configuration", key)
		}
	}
	host := strings.Trim(strings.TrimSpace(os.Getenv("DATABASE_HOST")), "[]")
	if host == "" || strings.ContainsAny(host, "/?#@ \t\r\n") || (strings.Contains(host, ":") && net.ParseIP(host) == nil) {
		return "", errors.New("DATABASE_HOST must be a hostname or IP address without a port")
	}
	port, err := strconv.Atoi(getenv("DATABASE_PORT", "1433"))
	if err != nil || port < 1 || port > 65535 {
		return "", errors.New("DATABASE_PORT must be in 1..65535")
	}
	encrypt := strings.ToLower(strings.TrimSpace(getenv("DATABASE_ENCRYPT", "disable")))
	switch encrypt {
	case "disable", "false", "true":
	default:
		return "", errors.New("DATABASE_ENCRYPT must be disable, false or true")
	}
	dsn := url.URL{
		Scheme: "sqlserver",
		Host:   net.JoinHostPort(host, strconv.Itoa(port)),
		User:   url.UserPassword(os.Getenv("DATABASE_USERNAME"), os.Getenv("DATABASE_PASSWORD")),
	}
	query := url.Values{}
	query.Set("database", os.Getenv("DATABASE_NAME"))
	query.Set("encrypt", encrypt)
	dsn.RawQuery = query.Encode()
	return dsn.String(), nil
}

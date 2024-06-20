package ollama

import (
	"errors"
	"net"
	"os"
	"strconv"
	"strings"
)

var ErrInvalidHostPort = errors.New("invalid port specified in OLLAMA_HOST")

type OllamaHost struct {
	Scheme string
	Host   string
	Port   string
}

// implementation of the function getOllamaHost() from ollama
//
// see https://github.com/ollama/ollama/blob/fedf71635ec77644f8477a86c6155217d9213a11/envconfig/config.go#L304
func getOllamaHost() (*OllamaHost, error) {
	defaultPort := "11434"

	hostVar := os.Getenv("OLLAMA_HOST")
	hostVar = strings.TrimSpace(strings.Trim(strings.TrimSpace(hostVar), "\"'"))

	scheme, hostport, ok := strings.Cut(hostVar, "://")
	switch {
	case !ok:
		scheme, hostport = "http", hostVar
	case scheme == "http":
		defaultPort = "80"
	case scheme == "https":
		defaultPort = "443"
	}

	// trim trailing slashes
	hostport = strings.TrimRight(hostport, "/")

	host, port, err := net.SplitHostPort(hostport)
	if err != nil {
		host, port = "127.0.0.1", defaultPort
		if ip := net.ParseIP(strings.Trim(hostport, "[]")); ip != nil {
			host = ip.String()
		} else if hostport != "" {
			host = hostport
		}
	}

	if portNum, err := strconv.ParseInt(port, 10, 32); err != nil || portNum > 65535 || portNum < 0 {
		return &OllamaHost{
			Scheme: scheme,
			Host:   host,
			Port:   defaultPort,
		}, ErrInvalidHostPort
	}

	return &OllamaHost{
		Scheme: scheme,
		Host:   host,
		Port:   port,
	}, nil
}

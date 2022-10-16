package shard

import (
	"os"
	"strconv"
)

const defaultServerHost = "localhost"
const defaultServerPort = 8080

// Application configuration.
type Config struct {
	Host string
	Port int
}

// Loads application configuration from environment variables and applies the
// default values specified as an argument if a variable doesn't present.
// The argument may be nil.
func NewConfigFromEnv(defaults *Config) *Config {
	host := os.Getenv("REC_HOST")
	if host == "" {
		if defaults != nil {
			host = defaults.Host
		} else {
			host = defaultServerHost
		}
	}
	port, err := strconv.Atoi(os.Getenv("REC_PORT"))
	if err != nil || port < 0 {
		if defaults != nil {
			port = defaults.Port
		} else {
			port = defaultServerPort
		}
	}
	return &Config{
		Host: host,
		Port: port,
	}
}

// Returns host and port as a string joined with a colon e.g. "localhost:8080".
func (c *Config) GetHostPort() string {
	return c.Host + ":" + strconv.Itoa(c.Port)
}

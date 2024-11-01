package config

import "time"

type HTTPServer struct {
	Host                    string
	Port                    int
	GracefulShutdownTimeout time.Duration
}

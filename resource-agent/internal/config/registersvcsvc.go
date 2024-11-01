package config

import "time"

type RegisterSvc struct {
	Host    string
	Port    int
	Timeout time.Duration
}

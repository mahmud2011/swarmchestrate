package config

import "time"

type LeaderSvc struct {
	Port    int
	Timeout time.Duration
}

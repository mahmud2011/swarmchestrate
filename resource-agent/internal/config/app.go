package config

import "time"

type App struct {
	Type              string
	HeartBeatInterval time.Duration
	DeploymentTimeout time.Duration
}

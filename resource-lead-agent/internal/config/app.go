package config

import "time"

type App struct {
	Type                        string
	SchedulerInterval           time.Duration
	ApplicationHeartbeatTimeout time.Duration
	NodeHeartbeatTimeout        time.Duration
}

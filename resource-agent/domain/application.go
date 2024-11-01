package domain

import (
	"errors"
)

type ApplicationState string

const (
	StateProgressing ApplicationState = "progressing"
	StateHealthy     ApplicationState = "healthy"
	StateFailed      ApplicationState = "failed"
	StateNULL        ApplicationState = "NULL"
)

var (
	ErrApplicationNotFound  error = errors.New("application not found")
	ErrApplicationNotSynced error = errors.New("application not synced")
)

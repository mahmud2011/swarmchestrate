package server

import "context"

type IServer interface {
	Start() error
	Stop(context.Context) error
}

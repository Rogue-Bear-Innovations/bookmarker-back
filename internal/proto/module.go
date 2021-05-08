package proto

import (
	"go.uber.org/fx"
)

var (
	Module = fx.Provide(
		NewGRPCServer,
	)
)

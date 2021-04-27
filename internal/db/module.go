package db

import (
	"go.uber.org/fx"
)

var (
	Module = fx.Provide(
		NewGormClient,
	)
)

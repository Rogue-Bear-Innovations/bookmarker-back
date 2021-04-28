package main

import (
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/Rogue-Bear-Innovations/bookmarker-back/internal/config"
	"github.com/Rogue-Bear-Innovations/bookmarker-back/internal/db"
	"github.com/Rogue-Bear-Innovations/bookmarker-back/internal/transport"
)

func main() {
	app := fx.New(
		transport.Module,
		db.Module,
		config.Module,
		fx.Provide(
			func() (*zap.SugaredLogger, error) {
				l, err := zap.NewProduction()
				if err != nil {
					return nil, err
				}
				s := l.Sugar()
				return s, nil
			},
		),
		fx.Invoke(func(server *transport.HTTPServer) {

		}),
	)

	app.Run()
}

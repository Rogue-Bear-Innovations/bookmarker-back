package main

import (
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/Rogue-Bear-Innovations/bookmarker-back/internal/config"
	"github.com/Rogue-Bear-Innovations/bookmarker-back/internal/db"
	"github.com/Rogue-Bear-Innovations/bookmarker-back/internal/transport"
)

type (
	Config struct {
		Host       string `mapstructure:"HOST"`
		Port       string `mapstructure:"PORT"`
		DBHost     string `mapstructure:"DB_HOST"`
		DBPort     string `mapstructure:"DB_PORT"`
		DBUser     string `mapstructure:"DB_USER"`
		DBPassword string `mapstructure:"DB_PASSWORD"`
		DBName     string `mapstructure:"DB_NAME"`
	}
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
				return l.Sugar(), nil
			},
		),
		fx.Invoke(func(server *transport.HTTPServer) {

		}),
	)

	app.Run()
}

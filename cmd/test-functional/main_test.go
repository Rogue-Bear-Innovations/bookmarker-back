package test_functional

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/jackc/pgx/v4"
	"github.com/spf13/viper"
)

var (
	DBConn     *pgx.Conn
	AppBaseURL url.URL
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

func TestMain(m *testing.M) {
	viper.SetEnvPrefix("TEST_RUNNER")

	viper.SetDefault("HOST", "0.0.0.0")
	viper.SetDefault("PORT", "1323")
	viper.SetDefault("DB_HOST", "0.0.0.0")
	viper.SetDefault("DB_PORT", "5432")
	viper.SetDefault("DB_USER", "user")
	viper.SetDefault("DB_PASSWORD", "password")
	viper.SetDefault("DB_NAME", "db")

	envs := []string{"HOST", "PORT", "DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME"}
	for _, key := range envs {
		if err := viper.BindEnv(key); err != nil {
			panic(err)
		}
	}

	cfg := Config{}
	if err := viper.Unmarshal(&cfg); err != nil {
		panic(err)
	}
	fmt.Println(cfg)

	////////

	pingCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)

	cl := resty.New()
	pingUrl := url.URL{
		Scheme: "http",
		Host:   cfg.Host + ":" + cfg.Port,
		Path:   "/ping",
	}
	pingUrlStr := pingUrl.String()
	for {
		if pingCtx.Err() != nil {
			panic(pingCtx.Err())
		}
		resp, err := cl.R().Get(pingUrlStr)
		if err != nil {
			panic(err)
		}
		if resp.String() == "pong" {
			break
		}
	}
	cancel()

	fmt.Println("pinged successfully")

	///////

	AppBaseURL = url.URL{
		Scheme: "http",
		Host:   cfg.Host + ":" + cfg.Port,
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBPort)
	conn, err := pgx.Connect(context.Background(), dsn)
	if err != nil {
		panic(err)
	}
	DBConn = conn

	/////////

	os.Exit(m.Run())
}

func FlushDB() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	if _, err := DBConn.Exec(ctx, "DELETE from tag_bookmarks"); err != nil {
		panic(err)
	}
	if _, err := DBConn.Exec(ctx, "DELETE from bookmarks"); err != nil {
		panic(err)
	}
	if _, err := DBConn.Exec(ctx, "DELETE from tags"); err != nil {
		panic(err)
	}
	if _, err := DBConn.Exec(ctx, "DELETE from users"); err != nil {
		panic(err)
	}
}

package db

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/pkg/errors"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/Rogue-Bear-Innovations/bookmarker-back/internal/config"
	"github.com/Rogue-Bear-Innovations/bookmarker-back/internal/models"
)

func NewGormClient(cfg *config.Config) (*gorm.DB, error) {
	newLogger := logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), logger.Config{
		SlowThreshold:             200 * time.Millisecond,
		LogLevel:                  logger.Info,
		Colorful:                  true,
		IgnoreRecordNotFoundError: false,
	})

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBPort)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect database")
	}

	if err := db.AutoMigrate(&models.User{}); err != nil {
		return nil, errors.Wrap(err, "migrate user")
	}
	if err := db.AutoMigrate(&models.Bookmark{}); err != nil {
		return nil, errors.Wrap(err, "migrate bookmark")
	}
	if err := db.AutoMigrate(&models.Tag{}); err != nil {
		return nil, errors.Wrap(err, "migrate tag")
	}

	return db, nil
}

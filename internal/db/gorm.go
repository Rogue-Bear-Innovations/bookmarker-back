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
)

type (
	GormForkedModel struct {
		ID        uint64 `gorm:"primarykey"`
		CreatedAt time.Time
		UpdatedAt time.Time
	}

	User struct {
		GormForkedModel
		Email     string `gorm:"unique;not null"`
		Password  string `gorm:"not null"`
		Token     string `gorm:"not null"`
		Bookmarks []Bookmark
		Tags      []Tag
	}

	Bookmark struct {
		GormForkedModel
		Name        *string
		Link        *string
		Description *string
		UserID      uint64 `gorm:"not null"`
		User        User
		Tags        []Tag `gorm:"many2many:tag_bookmarks;"`
	}

	Tag struct {
		GormForkedModel
		Name      string     `gorm:"not null;uniqueIndex:uidx_name_user_id"`
		Bookmarks []Bookmark `gorm:"many2many:tag_bookmarks;"`
		UserID    uint64     `gorm:"not null;uniqueIndex:uidx_name_user_id"`
		User      User
	}
)

func NewGormClient(cfg *config.Config) (*gorm.DB, error) {
	newLogger := logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), logger.Config{
		SlowThreshold:             200 * time.Millisecond,
		LogLevel:                  logger.Info,
		Colorful:                  true,
		IgnoreRecordNotFoundError: false,
	})

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBPort, cfg.DBSSLMode)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect database")
	}

	if err := db.AutoMigrate(&User{}); err != nil {
		return nil, errors.Wrap(err, "migrate user")
	}
	if err := db.AutoMigrate(&Bookmark{}); err != nil {
		return nil, errors.Wrap(err, "migrate bookmark")
	}
	if err := db.AutoMigrate(&Tag{}); err != nil {
		return nil, errors.Wrap(err, "migrate tag")
	}

	return db, nil
}

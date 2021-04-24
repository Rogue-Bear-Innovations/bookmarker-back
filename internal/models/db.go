package models

import (
	"gorm.io/gorm"
)

type (
	User struct {
		gorm.Model
		Email     string `gorm:"unique;not null"`
		Password  string `gorm:"not null"`
		Token     string `gorm:"not null"`
		Bookmarks []Bookmark
		Tags      []Tag
	}

	Bookmark struct {
		gorm.Model
		Name        *string
		Link        *string
		Description *string
		UserID      uint `gorm:"not null"`
		User        User
		Tags        []Tag `gorm:"many2many:tag_bookmarks;"`
	}

	Tag struct {
		gorm.Model
		Name      string     `gorm:"not null;uniqueIndex:uidx_name_user_id"`
		Bookmarks []Bookmark `gorm:"many2many:tag_bookmarks;"`
		UserID    uint64     `gorm:"not null;uniqueIndex:uidx_name_user_id"`
		User      User
	}
)

package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/gommon/log"
	"github.com/pkg/errors"

	"github.com/go-playground/validator"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type (
	CustomValidator struct {
		validator *validator.Validate
	}

	UserReq struct {
		Email string `json:"email" validate:"required,email"`
	}

	User struct {
		gorm.Model
		Email    string `gorm:"unique;not null"`
		Password string `gorm:"not null"`
		Token    string `gorm:"not null"`
	}

	Bookmark struct {
		gorm.Model
		Name        string
		Link        string
		Description string
	}

	Tag struct {
		gorm.Model
		Name      string     `gorm:"not null"`
		Bookmarks []Bookmark `gorm:"many2many:tag_bookmarks;"`
	}
)

func (cv *CustomValidator) Validate(i interface{}) error {
	if err := cv.validator.Struct(i); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return nil
}

func main() {
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// Migrate the schema
	if err := db.AutoMigrate(&Bookmark{}); err != nil {
		panic(err)
	}
	if err := db.AutoMigrate(&User{}); err != nil {
		panic(err)
	}
	if err := db.AutoMigrate(&Tag{}); err != nil {
		panic(err)
	}

	////////

	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}

	e.POST("/auth/register", func(c echo.Context) error {
		u := UserReq{}
		if err = c.Bind(&u); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		if err = c.Validate(&u); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		fmt.Println(u)
		token := uuid.New().String()
		res := db.Create(&User{
			Email: u.Email,
			Token: token,
		})
		if res.Error != nil {
			return res.Error
		}
		resp := struct {
			Token string `json:"token"`
		}{
			Token: token,
		}
		return c.JSON(http.StatusOK, &resp)
	})

	//bookmarkG := e.Group("/bookmark")
	//bookmarkG.GET("", func(c echo.Context) error {
	//
	//})
	//bookmarkG.POST("", func(c echo.Context) error {
	//
	//})
	//bookmarkG.DELETE("/:id", func(c echo.Context) error {
	//
	//})

	e.Use(middleware.CORS())
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	echo.NotFoundHandler = func(c echo.Context) error {
		return c.NoContent(http.StatusNotFound)
	}
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Path() == "/auth/register" {
				return next(c)
			}
			token := ""
			for key, values := range c.Request().Header {
				if strings.ToLower(key) == "x-token" {
					token = values[0]
					break
				}
			}
			if token == "" {
				return c.NoContent(http.StatusUnauthorized)
			}
			user := User{}
			res := db.Where("token = ?", token).First(&user)
			if res.Error != nil {
				log.Error(errors.Wrap(err, "find user in db"))
				return c.NoContent(http.StatusUnauthorized)
			}

			c.Set("user", &user)
			return next(c)
		}
	})
	e.Logger.Fatal(e.Start(":1323"))
}

package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/viper"

	"github.com/go-playground/validator"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type (
	CustomValidator struct {
		validator *validator.Validate
	}

	UserReq struct {
		Email string `json:"email" validate:"required,email"`
	}

	BookmarkReq struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		Link        *string `json:"link"`
	}

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
	}

	Tag struct {
		gorm.Model
		Name      string     `gorm:"not null;uniqueIndex:uidx_name_user_id"`
		Bookmarks []Bookmark `gorm:"many2many:tag_bookmarks;"`
		UserID    uint64     `gorm:"not null;uniqueIndex:uidx_name_user_id"`
		User      User
	}

	Config struct {
		Host       string `mapstructure:"HOST"`
		Port       string `mapstructure:"PORT"`
		DBHost     string `mapstructure:"DB_HOST"`
		DBPort     string `mapstructure:"DB_PORT"`
		DBUser     string `mapstructure:"DB_USER"`
		DBPassword string `mapstructure:"DB_PASSWORD"`
		DBName     string `mapstructure:"DB_NAME"`
	}

	BookmarkResp struct {
		ID          uint    `json:"id"`
		Name        *string `json:"name,omitempty"`
		Link        *string `json:"link,omitempty"`
		Description *string `json:"description,omitempty"`
	}
)

func (cv *CustomValidator) Validate(i interface{}) error {
	if err := cv.validator.Struct(i); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return nil
}

func main() {
	viper.SetEnvPrefix("BOOKMARKER")

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

	/////////

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBPort)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	if err := db.AutoMigrate(&User{}); err != nil {
		panic(err)
	}
	if err := db.AutoMigrate(&Bookmark{}); err != nil {
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
		if err := BindAndValidate(c, &u); err != nil {
			return err
		}

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

	bookmarkG := e.Group("/bookmark")
	bookmarkG.POST("/list", func(c echo.Context) error {
		user, err := GetUserFromContext(c)
		if err != nil {
			return err
		}

		bookmarks := make([]Bookmark, 0)
		res := db.Where("user_id = ?", user.ID).Find(&bookmarks)
		if res.Error != nil {
			return res.Error
		}

		resp := make([]BookmarkResp, len(bookmarks))
		for i := range bookmarks {
			resp[i] = BookmarkResp{
				ID:          bookmarks[i].ID,
				Name:        bookmarks[i].Name,
				Link:        bookmarks[i].Link,
				Description: bookmarks[i].Description,
			}
		}
		return c.JSON(http.StatusOK, resp)
	})
	bookmarkG.POST("", func(c echo.Context) error {
		user, err := GetUserFromContext(c)
		if err != nil {
			return err
		}

		req := BookmarkReq{}
		if err := BindAndValidate(c, &req); err != nil {
			return err
		}

		model := Bookmark{
			Name:        req.Name,
			Link:        req.Link,
			Description: req.Description,
			UserID:      user.ID,
		}

		res := db.Create(model)
		if res.Error != nil {
			return res.Error
		}

		return c.JSON(http.StatusOK, BookmarkResp{
			ID:          model.ID,
			Name:        model.Name,
			Link:        model.Link,
			Description: model.Description,
		})
	})
	//bookmarkG.DELETE("/:id", func(c echo.Context) error {
	//
	//})

	e.GET("/ping", func(c echo.Context) error { return c.String(http.StatusOK, "pong") })

	e.Use(middleware.CORS())
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	echo.NotFoundHandler = func(c echo.Context) error {
		return c.NoContent(http.StatusNotFound)
	}
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Path() == "/auth/register" || c.Path() == "/ping" {
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
				c.Logger().Error(errors.Wrap(err, "find user in db"))
				return c.NoContent(http.StatusUnauthorized)
			}

			c.Set("user", &user)
			return next(c)
		}
	})
	listen := cfg.Host + ":" + cfg.Port
	e.Logger.Fatal(e.Start(listen))
}

func BindAndValidate(c echo.Context, v interface{}) error {
	var err error
	if err = c.Bind(v); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err = c.Validate(v); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return nil
}

func GetUserFromContext(c echo.Context) (*User, error) {
	user := c.Get("user").(*User)
	if user == nil {
		return nil, errors.New("no user found in context")
	}
	return user, nil
}

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4/middleware"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"gorm.io/gorm/logger"

	"github.com/go-playground/validator"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/Rogue-Bear-Innovations/bookmarker-back/internal/models"
)

type (
	CustomValidator struct {
		validator *validator.Validate
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
		panic("failed to connect database")
	}

	if err := db.AutoMigrate(&models.User{}); err != nil {
		panic(err)
	}
	if err := db.AutoMigrate(&models.Bookmark{}); err != nil {
		panic(err)
	}
	if err := db.AutoMigrate(&models.Tag{}); err != nil {
		panic(err)
	}

	////////

	e := echo.New()

	e.POST("/auth/register", func(c echo.Context) error {
		u := models.UserReq{}
		if err := BindAndValidate(c, &u); err != nil {
			return err
		}

		token := uuid.New().String()
		res := db.Create(&models.User{
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

		bookmarks := make([]models.Bookmark, 0)
		res := db.Where("user_id = ?", user.ID).Find(&bookmarks)
		if res.Error != nil {
			return res.Error
		}

		resp := make([]models.BookmarkResp, len(bookmarks))
		for i := range bookmarks {
			resp[i] = models.BookmarkResp{
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

		req := models.BookmarkReq{}
		if err := BindAndValidate(c, &req); err != nil {
			return err
		}

		model := models.Bookmark{
			Name:        req.Name,
			Link:        req.Link,
			Description: req.Description,
			UserID:      user.ID,
		}

		res := db.Create(&model)
		if res.Error != nil {
			return res.Error
		}

		return c.JSON(http.StatusOK, models.BookmarkResp{
			ID:          model.ID,
			Name:        model.Name,
			Link:        model.Link,
			Description: model.Description,
		})
	})
	bookmarkG.PATCH("/:id", func(c echo.Context) error {
		id, err := GetAndParseParam(c, "id")
		if err != nil {
			return err
		}
		user, err := GetUserFromContext(c)
		if err != nil {
			return err
		}

		req := models.BookmarkReq{}
		if err := BindAndValidate(c, &req); err != nil {
			return err
		}

		model := models.Bookmark{
			Model: gorm.Model{
				ID: uint(id),
			},
			Name:        req.Name,
			Link:        req.Link,
			Description: req.Description,
			UserID:      user.ID,
		}

		res := db.Model(&model).Updates(&model)
		if res.Error != nil {
			return res.Error
		}

		return c.JSON(http.StatusOK, models.BookmarkResp{
			ID:          model.ID,
			Name:        model.Name,
			Link:        model.Link,
			Description: model.Description,
		})
	})
	bookmarkG.DELETE("/:id", func(c echo.Context) error {
		id := c.Param("id")
		if id == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid path param 'id'")
		}
		res := db.Delete(&models.Bookmark{}, id)
		if res.Error != nil {
			return res.Error
		}
		return c.NoContent(http.StatusNoContent)
	})

	tagG := e.Group("/tag")
	tagG.GET("", func(c echo.Context) error {
		user, err := GetUserFromContext(c)
		if err != nil {
			return err
		}

		tags := make([]models.Tag, 0)
		res := db.Where("user_id = ?", user.ID).Find(&tags)
		if res.Error != nil {
			return res.Error
		}

		resp := make([]models.TagResp, len(tags))
		for i := range tags {
			resp[i] = models.TagResp{
				ID:   tags[i].ID,
				Name: tags[i].Name,
			}
		}
		return c.JSON(http.StatusOK, resp)
	})
	tagG.POST("", func(c echo.Context) error {
		user, err := GetUserFromContext(c)
		if err != nil {
			return err
		}

		req := models.TagReq{}
		if err := BindAndValidate(c, &req); err != nil {
			return err
		}

		model := models.Tag{
			Name:   req.Name,
			UserID: uint64(user.ID),
		}

		res := db.Create(&model)
		if res.Error != nil {
			return res.Error
		}

		return c.JSON(http.StatusOK, models.TagResp{
			ID:   model.ID,
			Name: model.Name,
		})
	})
	tagG.PATCH("/:id", func(c echo.Context) error {
		id, err := GetAndParseParam(c, "id")
		if err != nil {
			return err
		}
		user, err := GetUserFromContext(c)
		if err != nil {
			return err
		}

		req := models.TagReq{}
		if err := BindAndValidate(c, &req); err != nil {
			return err
		}

		model := models.Tag{
			Model: gorm.Model{
				ID: uint(id),
			},
			Name:   req.Name,
			UserID: uint64(user.ID),
		}

		res := db.Model(&model).Updates(&model)
		if res.Error != nil {
			return res.Error
		}

		return c.JSON(http.StatusOK, models.TagResp{
			ID:   model.ID,
			Name: model.Name,
		})
	})
	tagG.DELETE("/:id", func(c echo.Context) error {
		id := c.Param("id")
		if id == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid path param 'id'")
		}
		res := db.Delete(&models.Tag{}, id)
		if res.Error != nil {
			return res.Error
		}
		return c.NoContent(http.StatusNoContent)
	})

	e.GET("/ping", func(c echo.Context) error { return c.String(http.StatusOK, "pong") })

	e.Use(middleware.CORS())
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Validator = &CustomValidator{validator: validator.New()}
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
			user := models.User{}
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

func GetUserFromContext(c echo.Context) (*models.User, error) {
	user := c.Get("user").(*models.User)
	if user == nil {
		return nil, errors.New("no user found in context")
	}
	return user, nil
}

func GetParam(c echo.Context, name string) (string, error) {
	value := c.Param(name)
	if value == "" {
		return "", echo.NewHTTPError(http.StatusBadRequest, "invalid path param 'id'")
	}
	return value, nil
}

func GetAndParseParam(c echo.Context, name string) (uint64, error) {
	v, e := GetParam(c, name)
	if e != nil {
		return 0, e
	}
	vv, e := strconv.ParseUint(v, 10, 64)
	if e != nil {
		return 0, e
	}
	return vv, nil
}

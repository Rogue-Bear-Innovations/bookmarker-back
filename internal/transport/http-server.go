package transport

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/go-playground/validator"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pkg/errors"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Rogue-Bear-Innovations/bookmarker-back/internal/config"
	"github.com/Rogue-Bear-Innovations/bookmarker-back/internal/db"
)

type (
	UserReq struct {
		Email string `json:"email" validate:"required,email"`
	}

	BookmarkReq struct {
		Name        *string  `json:"name"`
		Description *string  `json:"description"`
		Link        *string  `json:"link"`
		Tags        []uint64 `json:"tags"`
	}

	BookmarkReqList struct {
		Tags []uint64 `json:"tags"`
	}

	BookmarkResp struct {
		ID          uint    `json:"id"`
		Name        *string `json:"name,omitempty"`
		Link        *string `json:"link,omitempty"`
		Description *string `json:"description,omitempty"`
	}

	TagReq struct {
		Name string `json:"name" validate:"required"`
	}

	TagResp struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
	}

	CustomValidator struct {
		validator *validator.Validate
	}

	HTTPServer struct {
		db *gorm.DB
	}
)

func NewHTTPServer(lc fx.Lifecycle, cfg *config.Config, db *gorm.DB, logger *zap.SugaredLogger) *HTTPServer {
	e := echo.New()

	instance := HTTPServer{
		db: db,
	}

	e.POST("/auth/register", instance.Register)

	bookmarkG := e.Group("/bookmark")
	bookmarkG.POST("/list", instance.BookmarkGet)
	bookmarkG.POST("", instance.BookmarkCreate)
	bookmarkG.PATCH("/:id", instance.BookmarkUpdate)
	bookmarkG.DELETE("/:id", instance.BookmarkUpdate)

	tagG := e.Group("/tag")
	tagG.GET("", instance.TagGet)
	tagG.POST("", instance.TagCreate)
	tagG.PATCH("/:id", instance.TagUpdate)
	tagG.DELETE("/:id", instance.TagDelete)

	e.GET("/ping", func(c echo.Context) error { return c.String(http.StatusOK, "pong") })

	e.Use(middleware.CORS())
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.Use(instance.AuthMiddleware)

	e.Validator = &CustomValidator{validator: validator.New()}

	echo.NotFoundHandler = func(c echo.Context) error {
		return c.NoContent(http.StatusNotFound)
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				listen := cfg.Host + ":" + cfg.Port
				if err := e.Start(listen); err != nil && err != http.ErrServerClosed {
					e.Logger.Fatal("shutting down the server")
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping HTTP server.")
			return e.Shutdown(ctx)
		},
	})

	return &instance
}

func (s *HTTPServer) Register(c echo.Context) error {
	u := UserReq{}
	if err := BindAndValidate(c, &u); err != nil {
		return err
	}

	token := uuid.New().String()
	res := s.db.Create(&db.User{
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
}

func (s *HTTPServer) BookmarkGet(c echo.Context) error {
	user, err := GetUserFromContext(c)
	if err != nil {
		return err
	}

	req := BookmarkReqList{}
	if err := BindAndValidate(c, &req); err != nil {
		return err
	}

	w := squirrel.Eq{
		"b.user_id": user.ID,
	}
	if len(req.Tags) != 0 {
		w["tb.tag_id"] = req.Tags
	}
	sql, args, err := squirrel.
		Select("b.id", "b.link", "b.name", "b.description").From("bookmarks b").
		LeftJoin("tag_bookmarks tb ON b.id = tb.bookmark_id").
		OrderBy("b.id").
		Where(w).
		ToSql()
	if err != nil {
		return err
	}

	bookmarks := make([]db.Bookmark, 0)
	res := s.db.Raw(sql, args...).Scan(&bookmarks)
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
}

func (s *HTTPServer) BookmarkCreate(c echo.Context) error {
	user, err := GetUserFromContext(c)
	if err != nil {
		return err
	}

	req := BookmarkReq{}
	if err := BindAndValidate(c, &req); err != nil {
		return err
	}

	newTags := make([]db.Tag, len(req.Tags))
	for i := range req.Tags {
		newTags[i] = db.Tag{
			Model: gorm.Model{
				ID: uint(req.Tags[i]),
			},
		}
	}

	model := db.Bookmark{
		Name:        req.Name,
		Link:        req.Link,
		Description: req.Description,
		UserID:      user.ID,
		Tags:        newTags,
	}

	res := s.db.Create(&model)
	if res.Error != nil {
		return res.Error
	}

	return c.JSON(http.StatusOK, BookmarkResp{
		ID:          model.ID,
		Name:        model.Name,
		Link:        model.Link,
		Description: model.Description,
	})
}

func (s *HTTPServer) BookmarkUpdate(c echo.Context) error {
	id, err := GetAndParseParam(c, "id")
	if err != nil {
		return err
	}
	user, err := GetUserFromContext(c)
	if err != nil {
		return err
	}

	req := BookmarkReq{}
	if err := BindAndValidate(c, &req); err != nil {
		return err
	}

	newTags := make([]db.Tag, len(req.Tags))
	for i := range req.Tags {
		newTags[i] = db.Tag{
			Model: gorm.Model{
				ID: uint(req.Tags[i]),
			},
		}
	}

	model := db.Bookmark{
		Model: gorm.Model{
			ID: uint(id),
		},
		Name:        req.Name,
		Link:        req.Link,
		Description: req.Description,
		UserID:      user.ID,
		Tags:        newTags,
	}

	res := s.db.Model(&model).Updates(&model)
	if res.Error != nil {
		return res.Error
	}

	return c.JSON(http.StatusOK, BookmarkResp{
		ID:          model.ID,
		Name:        model.Name,
		Link:        model.Link,
		Description: model.Description,
	})
}

func (s *HTTPServer) BookmarkDelete(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid path param 'id'")
	}
	res := s.db.Delete(&db.Bookmark{}, id)
	if res.Error != nil {
		return res.Error
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *HTTPServer) TagGet(c echo.Context) error {
	user, err := GetUserFromContext(c)
	if err != nil {
		return err
	}

	tags := make([]db.Tag, 0)
	res := s.db.Where("user_id = ?", user.ID).Find(&tags)
	if res.Error != nil {
		return res.Error
	}

	resp := make([]TagResp, len(tags))
	for i := range tags {
		resp[i] = TagResp{
			ID:   tags[i].ID,
			Name: tags[i].Name,
		}
	}
	return c.JSON(http.StatusOK, resp)
}

func (s *HTTPServer) TagCreate(c echo.Context) error {
	user, err := GetUserFromContext(c)
	if err != nil {
		return err
	}

	req := TagReq{}
	if err := BindAndValidate(c, &req); err != nil {
		return err
	}

	model := db.Tag{
		Name:   req.Name,
		UserID: uint64(user.ID),
	}

	res := s.db.Create(&model)
	if res.Error != nil {
		return res.Error
	}

	return c.JSON(http.StatusOK, TagResp{
		ID:   model.ID,
		Name: model.Name,
	})
}

func (s *HTTPServer) TagUpdate(c echo.Context) error {
	id, err := GetAndParseParam(c, "id")
	if err != nil {
		return err
	}
	user, err := GetUserFromContext(c)
	if err != nil {
		return err
	}

	req := TagReq{}
	if err := BindAndValidate(c, &req); err != nil {
		return err
	}

	model := db.Tag{
		Model: gorm.Model{
			ID: uint(id),
		},
		Name:   req.Name,
		UserID: uint64(user.ID),
	}

	res := s.db.Model(&model).Updates(&model)
	if res.Error != nil {
		return res.Error
	}

	return c.JSON(http.StatusOK, TagResp{
		ID:   model.ID,
		Name: model.Name,
	})
}

func (s *HTTPServer) TagDelete(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid path param 'id'")
	}
	res := s.db.Delete(&db.Tag{}, id)
	if res.Error != nil {
		return res.Error
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *HTTPServer) AuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
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
		user := db.User{}
		res := s.db.Where("token = ?", token).First(&user)
		if res.Error != nil {
			c.Logger().Error(errors.Wrap(res.Error, "find user in db"))
			return c.NoContent(http.StatusUnauthorized)
		}

		c.Set("user", &user)
		return next(c)
	}
}

////////

func (cv *CustomValidator) Validate(i interface{}) error {
	if err := cv.validator.Struct(i); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return nil
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

func GetUserFromContext(c echo.Context) (*db.User, error) {
	user := c.Get("user").(*db.User)
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

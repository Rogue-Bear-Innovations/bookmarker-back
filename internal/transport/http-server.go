package transport

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/go-playground/validator"
	"github.com/gofiber/fiber/v2/middleware/cors"
	fiberLogger "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Rogue-Bear-Innovations/bookmarker-back/internal/config"
	"github.com/Rogue-Bear-Innovations/bookmarker-back/internal/db"

	"github.com/gofiber/fiber/v2"
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

	HTTPServer struct {
		db     *gorm.DB
		logger *zap.SugaredLogger
	}
)

func NewHTTPServer(lc fx.Lifecycle, cfg *config.Config, db *gorm.DB, logger *zap.SugaredLogger) *HTTPServer {
	app := fiber.New(fiber.Config{
		IdleTimeout: time.Second * 30,
		ErrorHandler: func(ctx *fiber.Ctx, err error) error {
			logger.Errorw("request failed",
				"error", err,
				"path", ctx.Path(),
				"method", ctx.Method(),
				"request_body", string(ctx.Body()),
				"request_headers", ctx.Request().Header.String(),
				"request_query", string(ctx.Request().URI().QueryString()),
			)
			return fiber.DefaultErrorHandler(ctx, err)
		},
	})

	instance := HTTPServer{
		db:     db,
		logger: logger,
	}

	// middlewares
	app.Use(cors.New())        // might add options https://github.com/gofiber/fiber/tree/master/middleware/cors
	app.Use(fiberLogger.New()) // https://github.com/gofiber/fiber/blob/master/middleware/logger/README.md
	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
	})) // https://github.com/gofiber/fiber/blob/master/middleware/recover/README.md

	// routes
	app.Get("/ping", func(c *fiber.Ctx) error { return c.SendString("pong") })

	authG := app.Group("/auth")
	authG.Post("/register", instance.Register)

	internalG := app.Group("")

	internalG.Use(instance.AuthMiddleware)

	bookmarkG := internalG.Group("/bookmark")
	bookmarkG.Post("/list", instance.BookmarkGet)
	bookmarkG.Post("", instance.BookmarkCreate)
	bookmarkG.Patch("/:id", instance.BookmarkUpdate)
	bookmarkG.Delete("/:id", instance.BookmarkDelete)

	tagG := internalG.Group("/tag")
	tagG.Get("", instance.TagGet)
	tagG.Post("", instance.TagCreate)
	tagG.Patch("/:id", instance.TagUpdate)
	tagG.Delete("/:id", instance.TagDelete)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				listen := cfg.Host + ":" + cfg.Port
				if err := app.Listen(listen); err != nil {
					logger.Fatalw("server stopped", "error", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping HTTP server.")
			return app.Shutdown()
		},
	})

	return &instance
}

func (s *HTTPServer) AuthMiddleware(c *fiber.Ctx) error {
	fmt.Println("running middleware")
	token := c.Get("x-token")
	if token == "" {
		return c.SendStatus(http.StatusUnauthorized)
	}

	user := db.User{}
	res := s.db.Where("token = ?", token).First(&user)
	if res.Error != nil {
		return c.SendStatus(http.StatusUnauthorized)
	}

	c.Locals("user", &user)
	return c.Next()
}

func (s *HTTPServer) Register(c *fiber.Ctx) error {
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
	return c.JSON(&resp)
}

func (s *HTTPServer) BookmarkGet(c *fiber.Ctx) error {
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
	return c.JSON(resp)
}

func (s *HTTPServer) BookmarkCreate(c *fiber.Ctx) error {
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
			GormForkedModel: db.GormForkedModel{
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

	return c.JSON(BookmarkResp{
		ID:          model.ID,
		Name:        model.Name,
		Link:        model.Link,
		Description: model.Description,
	})
}

func (s *HTTPServer) BookmarkUpdate(c *fiber.Ctx) error {
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
			GormForkedModel: db.GormForkedModel{
				ID: uint(req.Tags[i]),
			},
		}
	}

	model := db.Bookmark{
		GormForkedModel: db.GormForkedModel{
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

	return c.JSON(BookmarkResp{
		ID:          model.ID,
		Name:        model.Name,
		Link:        model.Link,
		Description: model.Description,
	})
}

func (s *HTTPServer) BookmarkDelete(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(http.StatusBadRequest).SendString("invalid path param 'id'")
	}
	res := s.db.Delete(&db.Bookmark{}, id)
	if res.Error != nil {
		return res.Error
	}
	return c.SendStatus(http.StatusNoContent)
}

func (s *HTTPServer) TagGet(c *fiber.Ctx) error {
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
	return c.JSON(resp)
}

func (s *HTTPServer) TagCreate(c *fiber.Ctx) error {
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

	return c.JSON(TagResp{
		ID:   model.ID,
		Name: model.Name,
	})
}

func (s *HTTPServer) TagUpdate(c *fiber.Ctx) error {
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
		GormForkedModel: db.GormForkedModel{
			ID: uint(id),
		},
		Name:   req.Name,
		UserID: uint64(user.ID),
	}

	res := s.db.Model(&model).Updates(&model)
	if res.Error != nil {
		return res.Error
	}

	return c.JSON(TagResp{
		ID:   model.ID,
		Name: model.Name,
	})
}

func (s *HTTPServer) TagDelete(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(http.StatusBadRequest).SendString("invalid path param 'id'")
	}
	res := s.db.Delete(&db.Tag{}, id)
	if res.Error != nil {
		return res.Error
	}
	return c.SendStatus(http.StatusNoContent)
}

////////

type ErrorResponse struct {
	FailedField string
	Tag         string
	Value       string
}

func ValidateStruct(v interface{}) []*ErrorResponse {
	var errs []*ErrorResponse
	validate := validator.New()
	err := validate.Struct(v)
	if err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			var element ErrorResponse
			element.FailedField = err.StructNamespace()
			element.Tag = err.Tag()
			element.Value = err.Param()
			errs = append(errs, &element)
		}
	}
	return errs
}

func BindAndValidate(c *fiber.Ctx, v interface{}) error {
	if err := c.BodyParser(v); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	errs := ValidateStruct(v)
	if errs != nil {
		return c.JSON(errs)
	}

	return nil
}

func GetUserFromContext(c *fiber.Ctx) (*db.User, error) {
	userRaw := c.Locals("user")
	if userRaw == nil {
		return nil, errors.New("no user found in context")
	}
	user, ok := userRaw.(*db.User)
	if !ok {
		return nil, errors.New("user context value conversion failed")
	}
	return user, nil
}

func GetParam(c *fiber.Ctx, name string) (string, error) {
	value := c.Params(name)
	if value == "" {
		return "", c.Status(http.StatusBadRequest).SendString("invalid path param 'id'")
	}
	return value, nil
}

func GetAndParseParam(c *fiber.Ctx, name string) (uint64, error) {
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

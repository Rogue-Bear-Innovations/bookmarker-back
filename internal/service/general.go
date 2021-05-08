package service

import (
	"github.com/Masterminds/squirrel"
	"github.com/Rogue-Bear-Innovations/bookmarker-back/internal/db"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrLoginUserNotFound         = errors.New("user not found")
	ErrLoginPasswordDoesNotMatch = errors.New("password does not match")
)

type General struct {
	db     *gorm.DB
	logger *zap.SugaredLogger
}

func NewGeneral(db *gorm.DB, l *zap.SugaredLogger) *General {
	return &General{
		db:     db,
		logger: l,
	}
}

func (s *General) Register(email, pass string) (string, error) {
	hash, err := s.bcryptGen(pass)
	if err != nil {
		return "", errors.Wrap(err, "bcryptGen")
	}
	token := uuid.New().String()
	res := s.db.Create(&db.User{
		Email:    email,
		Password: hash,
		Token:    token,
	})
	if res.Error != nil {
		return "", res.Error
	}
	return token, nil
}

func (s *General) Login(email, pass string) (string, error) {
	user := db.User{}
	res := s.db.Where("email = ?", email).First(&user)
	if res.Error != nil {
		if res.Error == gorm.ErrRecordNotFound {
			return "", ErrLoginUserNotFound
		}
		return "", res.Error
	}

	if err := s.bcryptCheck(user.Password, pass); err != nil {
		return "", ErrLoginPasswordDoesNotMatch
	}

	token := uuid.New().String()
	res = s.db.Model(&user).Update("token", token)
	if res.Error != nil {
		return "", errors.Wrap(res.Error, "update token")
	}

	return token, nil
}

func (s *General) BookmarkGet(user *db.User, tags []uint64) ([]db.Bookmark, error) {
	w := squirrel.Eq{
		"b.user_id": user.ID,
	}
	if len(tags) != 0 {
		w["tb.tag_id"] = tags
	}
	sql, args, err := squirrel.
		Select("b.id", "b.link", "b.name", "b.description").From("bookmarks b").
		LeftJoin("tag_bookmarks tb ON b.id = tb.bookmark_id").
		OrderBy("b.id").
		Where(w).
		ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "build sql")
	}

	bookmarks := make([]db.Bookmark, 0)
	res := s.db.Raw(sql, args...).Scan(&bookmarks)
	if res.Error != nil {
		return nil, errors.Wrap(res.Error, "scan")
	}

	return bookmarks, nil
}

func (s *General) BookmarkCreate(user *db.User, name, description, link *string, tagIds []uint64) (*db.Bookmark, error) {
	newTags := make([]db.Tag, len(tagIds))
	for i := range tagIds {
		newTags[i] = db.Tag{
			GormForkedModel: db.GormForkedModel{
				ID: tagIds[i],
			},
		}
	}

	model := db.Bookmark{
		Name:        name,
		Link:        link,
		Description: description,
		UserID:      user.ID,
		Tags:        newTags,
	}

	res := s.db.Create(&model)
	if res.Error != nil {
		return nil, res.Error
	}

	return &model, nil
}

func (s *General) BookmarkUpdate(user *db.User, bookmarkID uint64, tagIds []uint64, name, description, link *string) (*db.Bookmark, error) {
	newTags := make([]db.Tag, len(tagIds))
	for i := range tagIds {
		newTags[i] = db.Tag{
			GormForkedModel: db.GormForkedModel{
				ID: tagIds[i],
			},
		}
	}

	model := db.Bookmark{
		GormForkedModel: db.GormForkedModel{
			ID: bookmarkID,
		},
		Name:        name,
		Link:        link,
		Description: description,
		UserID:      user.ID,
		Tags:        newTags,
	}

	res := s.db.Model(&model).Updates(&model)
	if res.Error != nil {
		return nil, errors.Wrap(res.Error, "update model")
	}

	res = s.db.First(&model)
	if res.Error != nil {
		return nil, errors.Wrap(res.Error, "get model")
	}

	return &model, nil
}

func (s *General) BookmarkDelete(id uint64, user *db.User) error {
	res := s.db.Delete(&db.Bookmark{
		UserID: user.ID,
	}, id)
	if res.Error != nil {
		return res.Error
	}
	return nil
}

func (s *General) TagGet(userID uint64) ([]db.Tag, error) {
	tags := make([]db.Tag, 0)

	res := s.db.Where("user_id = ?", userID).Find(&tags)
	if res.Error != nil {
		return nil, res.Error
	}

	return tags, nil
}

func (s *General) TagCreate(userID uint64, name string) (*db.Tag, error) {
	model := db.Tag{
		Name:   name,
		UserID: userID,
	}

	res := s.db.Create(&model)
	if res.Error != nil {
		return nil, res.Error
	}

	return &model, nil
}

func (s *General) TagUpdate(tagID uint64, userID uint64, name string) (*db.Tag, error) {
	model := db.Tag{
		GormForkedModel: db.GormForkedModel{
			ID: tagID,
		},
		Name:   name,
		UserID: userID,
	}

	res := s.db.Model(&model).Updates(&model)
	if res.Error != nil {
		return nil, res.Error
	}

	return &model, nil
}

func (s *General) TagDelete(id, userID uint64) error {
	res := s.db.Delete(&db.Tag{
		UserID: userID,
	}, id)
	if res.Error != nil {
		return res.Error
	}
	return nil
}

func (s *General) bcryptGen(pass string) (string, error) {
	passwordHashB, err := bcrypt.GenerateFromPassword([]byte(pass), 14)
	if err != nil {
		return "", errors.Wrap(err, "generate password hash")
	}
	return string(passwordHashB), nil
}

func (s *General) bcryptCheck(hash, pass string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pass))
}

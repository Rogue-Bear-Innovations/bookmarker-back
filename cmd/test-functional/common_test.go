package test_functional

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
)

func TestRegister(t *testing.T) {
	u := AppBaseURL
	u.Path = "/auth/register"

	t.Run("successful register", func(t *testing.T) {
		defer FlushDB()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		type Resp struct {
			Token string `json:"token"`
		}

		resp, err := resty.New().
			R().
			SetHeader("Content-Type", "application/json").
			SetContext(ctx).
			SetResult(&Resp{}).
			SetBody(`
			{"email": "test@gmail.com", "password": "111111111111"}
		`).
			Post(u.String())
		assert.Nil(t, err)

		assert.Equal(t, http.StatusOK, resp.StatusCode())

		got, ok := resp.Result().(*Resp)
		assert.True(t, ok)
		assert.NotEmpty(t, got.Token)

		var (
			id    uint64
			token string
		)
		err = DBConn.QueryRow(ctx, "SELECT id, token FROM users WHERE token=$1", got.Token).Scan(&id, &token)
		assert.Nil(t, err)

		assert.Equal(t, token, got.Token)
	})

	t.Run("bad body", func(t *testing.T) {
		defer FlushDB()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		resp, err := resty.New().
			R().
			SetHeader("Content-Type", "application/json").
			SetContext(ctx).
			SetBody(`
			{"something": "???"}
		`).
			Post(u.String())
		assert.Nil(t, err)

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode())
	})
}

//
//func TestBookmarksCrud(t *testing.T) {
//	defer FlushDB()
//
//	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
//	defer cancel()
//
//	listURL := AppBaseURL
//	listURL.Path = "/bookmark/list"
//
//	//////
//
//	_, err := DBConn.Exec(ctx, "INSERT INTO users (id, token) VALUES (1, 'token')")
//	assert.Nil(t, err)
//	_, err = DBConn.Exec(ctx, "INSERT INTO bookmarks (name, description, link, user_id) VALUES ('name', 'desc', 'link', 1)")
//	_, err = DBConn.Exec(ctx, "INSERT INTO bookmarks (name, description, link, user_id) VALUES ('name', 'desc', 'link', 1)")
//	assert.Nil(t, err)
//
//	//////
//
//	resp, err := resty.New().
//		R().
//		SetHeader("Content-Type", "application/json").
//		SetContext(ctx).
//		SetResult(&[]models.BookmarkResp{}).
//		Get(listURL.String())
//	assert.Nil(t, err)
//
//	assert.Equal(t, http.StatusOK, resp.StatusCode())
//
//	n := "name"
//	d := "desc"
//	l := "link"
//	gotp, ok := resp.Result().(*[]models.BookmarkResp)
//	assert.True(t, ok)
//	got := *gotp
//	assert.Equal(t, []models.BookmarkResp{{
//		Name:        &n,
//		Link:        &d,
//		Description: &l,
//	},{
//		Name:        &n,
//		Link:        &d,
//		Description: &l,
//	}}, got)
//}

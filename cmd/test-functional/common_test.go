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
			{"email": "test@gmail.com"}
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

package transport

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCensorBody(t *testing.T) {
	b := `{
		"email": "email@email.com",
		"password": "123456789123"
	}`

	got := censorBody([]byte(b))
	assert.JSONEq(t, `{
		"email": "email@email.com",
		"password": "$censored"
	}`, string(got))
}

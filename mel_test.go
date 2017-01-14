package mel

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestCreateApp(t *testing.T) {
	r := New()
	assert.Equal(t, "/", r.BasePath)
	assert.Equal(t, r.Router, r.router)
	assert.Empty(t, r.Handlers)
}

package hashistack

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseConsulToken(t *testing.T) {

	token, err := parseConsulToken(filepath.Join("testdata", "bootstrap.txt"))
	assert.NoError(t, err)
	assert.Equal(t, "4456269a-e46a-c5bd-08d5-914552161f02", token)

}

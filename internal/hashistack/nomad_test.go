package hashistack

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNomad(t *testing.T) {
	nomad := NewNomadClient(".", "127.0.0.1", "cacert.pem", "cacert.crt", "cacert.key")
	assert.NotNil(t, nomad)
	cli := nomad.(*nomadCli)
	assert.Len(t, cli.Get(), 4)
	assert.Equal(t, cli.WorkingDir(), ".")

}

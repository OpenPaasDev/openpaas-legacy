package runtime

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommandExists(t *testing.T) {

	assert.True(t, commandExists("ls"))
	assert.False(t, commandExists("fofofo"))
}

func TestHasDependencies(t *testing.T) {

	assert.NotPanics(t, func() {
		HasDependencies()
	})

}

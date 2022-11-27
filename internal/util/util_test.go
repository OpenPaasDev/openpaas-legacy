package util

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

func TestRandString(t *testing.T) {

	theMap := make(map[string]string)

	for i := 0; i <= 30; i++ {
		str := RandString(20)
		if _, ok := theMap[str]; ok {
			assert.True(t, false)
		} else {
			theMap[str] = str
		}
	}

}

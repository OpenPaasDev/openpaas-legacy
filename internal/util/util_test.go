package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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

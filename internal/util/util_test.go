package util

import (
	"context"
	"strings"
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

func TestGetPublicIP(t *testing.T) {
	ip, err := GetPublicIP(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, strings.Count(ip, "."), 3)
}

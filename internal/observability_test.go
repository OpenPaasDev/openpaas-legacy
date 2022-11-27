package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenPaas/openpaas/internal/ansible"
	"github.com/OpenPaas/openpaas/internal/conf"
	"github.com/OpenPaas/openpaas/internal/util"
	"github.com/stretchr/testify/assert"
)

func TestMkObservabilityConfigs(t *testing.T) {
	folder := util.RandString(8)
	assert.NoError(t, os.MkdirAll(filepath.Join(folder, "secrets"), 0750))

	assert.NoError(t, os.MkdirAll(filepath.Join(folder, "consul"), 0750))
	defer func() {
		assert.NoError(t, os.RemoveAll(filepath.Join(folder)))
	}()
	mkSecrets(t, folder)
	inv, err := ansible.LoadInventory(filepath.Join("testdata", "inventory"))
	assert.NoError(t, err)
	consul := &MockConsul{}
	config, err := conf.Load(filepath.Join("testdata", "config.yaml"))
	config.BaseDir = folder
	assert.NoError(t, err)
	assert.NoError(t, mkObservabilityConfigs(consul, config, inv))

	assert.Equal(t, 6, len(consul.RegisterIntentionCalls()))
	assert.Equal(t, 6, len(consul.RegisterServiceCalls()))

	assert.Equal(t, 25, readDir(folder))
}

func readDir(str string) int {
	count := 0
	d, e := os.ReadDir(str)
	if e != nil {
		panic(e)
	}
	for _, f := range d {
		if f.IsDir() {
			count = count + readDir(filepath.Join(str, f.Name()))
		} else {
			fmt.Println(f.Name())
			count++
		}
	}
	return count
}

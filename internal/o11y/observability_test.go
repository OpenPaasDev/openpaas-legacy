package o11y

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenPaaSDev/openpaas/internal/ansible"
	"github.com/OpenPaaSDev/openpaas/internal/conf"
	"github.com/OpenPaaSDev/openpaas/internal/secrets"
	"github.com/OpenPaaSDev/openpaas/internal/util"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
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
	config, err := conf.Load(filepath.Join("..", "testdata", "config.yaml"))
	config.BaseDir = folder
	assert.NoError(t, err)
	assert.NoError(t, mkObservabilityConfigs(consul, config, inv))

	assert.Equal(t, 6, len(consul.RegisterIntentionCalls()))
	assert.Equal(t, 6, len(consul.RegisterServiceCalls()))

	assert.Equal(t, 26, readDir(folder))
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

func mkSecrets(t *testing.T, folder string) *secrets.Config { //nolint
	secrets := &secrets.Config{
		ConsulGossipKey:        "consulGossipKey",
		NomadGossipKey:         "nomadGossipKey",
		NomadClientConsulToken: "TBD",
		NomadServerConsulToken: "TBD",
		ConsulAgentToken:       "TBD",
		ConsulBootstrapToken:   "TBD",
		S3Endpoint:             "s3_endpoint_test",
		S3SecretKey:            "s3_secret_key_test",
		S3AccessKey:            "s3_access_key_test",
	}

	if _, err := os.Stat(filepath.Join(folder, "secrets", "secrets.yml")); errors.Is(err, os.ErrNotExist) {
		d, err := yaml.Marshal(secrets)
		assert.NoError(t, err)
		err = os.WriteFile(filepath.Join(folder, "secrets", "secrets.yml"), d, 0600)
		assert.NoError(t, err)
	}
	return secrets
}

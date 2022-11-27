package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenPaas/openpaas/internal/conf"
	"github.com/OpenPaas/openpaas/internal/secrets"
	sec "github.com/OpenPaas/openpaas/internal/secrets"
	"github.com/OpenPaas/openpaas/internal/util"
	"github.com/stretchr/testify/assert"
)

// func TestGenerateInventory(t *testing.T) {
// 	config, err := LoadConfig("testdata/config.yaml")
// 	assert.NoError(t, err)

// 	folder := RandString(8)
// 	config.BaseDir = folder
// 	err = os.MkdirAll(folder, 0700)
// 	assert.NoError(t, err)
// 	defer func() {
// 		e := os.RemoveAll(filepath.Join(folder))
// 		assert.NoError(t, e)
// 	}()

// 	src := filepath.Join("testdata", "inventory.json")
// 	dest := filepath.Join(folder, "inventory-output.json")

// 	bytesRead, err := os.ReadFile(filepath.Clean(src))
// 	assert.NoError(t, err)
// 	fmt.Println(string(bytesRead))

// 	err = os.WriteFile(filepath.Clean(dest), bytesRead, 0600)
// 	assert.NoError(t, err)

// 	err = GenerateInventory(config)
// 	assert.NoError(t, err)
// 	bytesRead, err = os.ReadFile(filepath.Clean(filepath.Join(folder, "inventory")))
// 	assert.NoError(t, err)
// 	assert.Equal(t, inventoryResultTest, string(bytesRead))
// }

func TestGenEnvRCFileExists(t *testing.T) {
	config := setUpEnvRCTest(t, true)

	err := GenerateEnvFile(config, config.BaseDir)
	assert.NoError(t, err)

	bytesRead, err := os.ReadFile(filepath.Clean(filepath.Join(config.BaseDir, ".envrc")))
	assert.NoError(t, err)
	str := string(bytesRead)
	assertContainsEnv(t, config, str)
	compare := `export S3_ENDPOINT=some-endpoint.com
export S3_ACCESS_KEY=access_key
export S3_SECRET_KEY=a_very_secret_key


### GENERATED CONFIG BELOW THIS LINE, DO NOT EDIT!`
	assert.Contains(t, str, compare)
	defer func() {
		e := os.RemoveAll(filepath.Join(config.BaseDir))
		assert.NoError(t, e)
	}()
}

func TestGenEnvRCFileDoesNotExist(t *testing.T) {
	config := setUpEnvRCTest(t, false)

	err := GenerateEnvFile(config, config.BaseDir)
	assert.NoError(t, err)

	bytesRead, err := os.ReadFile(filepath.Clean(filepath.Join(config.BaseDir, ".envrc")))
	assert.NoError(t, err)
	str := string(bytesRead)
	assertContainsEnv(t, config, str)
	defer func() {
		e := os.RemoveAll(filepath.Join(config.BaseDir))
		assert.NoError(t, e)
	}()
}

func assertContainsEnv(t *testing.T, config *conf.Config, str string) {
	assert.True(t, containsOneOf(func(s string) string { return fmt.Sprintf("export CONSUL_HTTP_ADDR=https://%s:8501", s) }, str))
	assert.Contains(t, str, "export CONSUL_HTTP_TOKEN=BootstrapToken")
	assert.Contains(t, str, fmt.Sprintf("export CONSUL_CLIENT_CERT=%s/secrets/consul/consul-agent-ca.pem", config.BaseDir))
	assert.Contains(t, str, fmt.Sprintf("export CONSUL_CLIENT_KEY=%s/secrets/consul/consul-agent-ca-key.pem", config.BaseDir))
	assert.True(t, containsOneOf(func(s string) string { return fmt.Sprintf("export NOMAD_ADDR=https://%s:4646", s) }, str))
	assert.Contains(t, str, fmt.Sprintf("export NOMAD_CACERT=%s/secrets/nomad/nomad-ca.pem", config.BaseDir))
	assert.Contains(t, str, fmt.Sprintf("export NOMAD_CLIENT_CERT=%s/secrets/nomad/client.pem", config.BaseDir))
	assert.Contains(t, str, fmt.Sprintf("export NOMAD_CLIENT_KEY=%s/secrets/nomad/client-key.pem", config.BaseDir))
	assert.True(t, containsOneOf(func(s string) string { return fmt.Sprintf("export VAULT_ADDR=https://%s:8200", s) }, str))
	assert.Contains(t, str, "export VAULT_SKIP_VERIFY=true")
}

func containsOneOf(format func(string) string, str string) bool {
	ips := []string{
		"127.0.0.1",
		"127.0.0.2",
		"127.0.0.3",
		"127.0.0.6",
		"127.0.0.7",
	}
	for _, ip := range ips {
		res := format(ip)
		if strings.Contains(str, res) {
			return true
		}
	}
	return false
}

func setUpEnvRCTest(t *testing.T, copyEnvRC bool) *conf.Config {
	config, err := conf.Load("testdata/config.yaml")
	assert.NoError(t, err)
	folder := util.RandString(8)
	config.BaseDir = folder
	err = os.MkdirAll(filepath.Join(folder, "secrets"), 0700)
	assert.NoError(t, err)

	copyTestFile(t, filepath.Join("testdata", "inventory"), filepath.Join(folder, "inventory"))
	if copyEnvRC {
		copyTestFile(t, filepath.Join("testdata", "envrc"), filepath.Join(folder, ".envrc"))
	}

	secrets := &secrets.Config{
		ConsulGossipKey:        "consulGossipKey",
		NomadGossipKey:         "nomadGossipKey",
		NomadClientConsulToken: "TBD",
		NomadServerConsulToken: "TBD",
		ConsulAgentToken:       "TBD",
		ConsulBootstrapToken:   "BootstrapToken",
		S3Endpoint:             "S3_ENDPOINT",
		S3SecretKey:            "S3_SECRET_KEY",
		S3AccessKey:            "S3_ACCESS_KEY",
	}

	err = sec.Write(folder, secrets)
	assert.NoError(t, err)
	return config
}

func copyTestFile(t *testing.T, src, dest string) {
	bytesRead, err := os.ReadFile(filepath.Clean(src))
	assert.NoError(t, err)

	err = os.WriteFile(filepath.Clean(dest), bytesRead, 0600)
	assert.NoError(t, err)
}

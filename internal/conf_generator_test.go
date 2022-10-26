package internal

import (
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hclparse"
	"github.com/stretchr/testify/assert"
	"github.com/zclconf/go-cty/cty"
)

type tfConfig struct {
	Variable []Variable `hcl:"variable,block"`
}

type Variable struct {
	Type      *hcl.Attribute `hcl:"type"`
	Name      string         `hcl:"name,label"`
	Default   *cty.Value     `hcl:"default,optional"`
	Sensitive bool           `hcl:"sensitive,optional"`
}

func TestGenerateTerraform(t *testing.T) {
	config, err := LoadConfig("testdata/config.yaml")
	assert.NoError(t, err)

	folder := RandString(8)
	config.BaseDir = folder
	defer func() {
		e := os.RemoveAll(filepath.Clean(folder))
		assert.NoError(t, e)
	}()

	err = GenerateTerraform(config, &CloudflareIPs{})
	assert.NoError(t, err)

	parser := hclparse.NewParser()
	f, parseDiags := parser.ParseHCLFile(filepath.Clean(filepath.Join(folder, "terraform", "vars.tf")))
	assert.False(t, parseDiags.HasErrors())

	_, parseDiags = parser.ParseHCLFile(filepath.Clean(filepath.Join(folder, "terraform", "main.tf")))
	assert.False(t, parseDiags.HasErrors())

	var conf tfConfig
	decodeDiags := gohcl.DecodeBody(f.Body, nil, &conf)
	assert.False(t, decodeDiags.HasErrors())

	vars := []struct {
		name       string
		tpe        string
		defaultVal cty.Value
	}{
		{name: "hcloud_token", tpe: "string", defaultVal: cty.NullVal(cty.String)},
		{name: "server_count", tpe: "number", defaultVal: cty.NumberVal(big.NewFloat(3))},
		{name: "client_count", tpe: "number", defaultVal: cty.NumberVal(big.NewFloat(2))},
		{name: "vault_count", tpe: "number", defaultVal: cty.NumberVal(big.NewFloat(2))},
		{name: "separate_consul_servers", tpe: "bool", defaultVal: cty.BoolVal(false)},
		{name: "multi_instance_observability", tpe: "bool", defaultVal: cty.BoolVal(false)},
		{name: "ssh_keys", tpe: "list", defaultVal: cty.TupleVal([]cty.Value{cty.StringVal("wille.faler@gmail.com")})},
		{name: "base_server_name", tpe: "string", defaultVal: cty.StringVal("nomad-srv")},
		{name: "firewall_name", tpe: "string", defaultVal: cty.StringVal("dev_firewall")},
		{name: "network_name", tpe: "string", defaultVal: cty.StringVal("dev_network")},
		{name: "allow_ips", tpe: "list", defaultVal: cty.TupleVal([]cty.Value{cty.StringVal("85.4.84.201/32")})},
		{name: "server_instance_type", tpe: "string", defaultVal: cty.StringVal("cx21")},
		{name: "location", tpe: "string", defaultVal: cty.StringVal("nbg1")},
	}

	expectedMap := make(map[string]string)
	for _, v := range conf.Variable {
		for _, expected := range vars {
			if expected.name == v.Name {
				expectedMap[expected.name] = expected.name
				assert.Equal(t, expected.tpe, v.Type.Expr.Variables()[0].RootName())
				if expected.defaultVal != cty.NullVal(cty.String) && !strings.Contains(expected.name, "_count") {
					assert.Equal(t, expected.defaultVal, *v.Default)
				}
				if strings.Contains(expected.name, "_count") {
					assert.Equal(t, expected.defaultVal.AsBigFloat().String(), v.Default.AsBigFloat().String())
				}
			}
		}
	}

	assert.Equal(t, len(expectedMap), len(vars))
}

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

func assertContainsEnv(t *testing.T, config *Config, str string) {
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

func setUpEnvRCTest(t *testing.T, copyEnvRC bool) *Config {
	config, err := LoadConfig("testdata/config.yaml")
	assert.NoError(t, err)
	folder := RandString(8)
	config.BaseDir = folder
	err = os.MkdirAll(filepath.Join(folder, "secrets"), 0700)
	assert.NoError(t, err)

	copyTestFile(t, filepath.Join("testdata", "inventory"), filepath.Join(folder, "inventory"))
	if copyEnvRC {
		copyTestFile(t, filepath.Join("testdata", "envrc"), filepath.Join(folder, ".envrc"))
	}

	secrets := &secretsConfig{
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

	err = writeSecrets(folder, secrets)
	assert.NoError(t, err)
	return config
}

func copyTestFile(t *testing.T, src, dest string) {
	bytesRead, err := os.ReadFile(filepath.Clean(src))
	assert.NoError(t, err)

	err = os.WriteFile(filepath.Clean(dest), bytesRead, 0600)
	assert.NoError(t, err)
}

package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func init() {
	if os.Getenv("S3_ENDPOINT") == "" {
		err := os.Setenv("S3_ENDPOINT", "aws.s3.amazon.com")
		if err != nil {
			panic(err)
		}
		err = os.Setenv("S3_SECRET_KEY", "aws.s3.amazon.com")
		if err != nil {
			panic(err)
		}
		err = os.Setenv("S3_ACCESS_KEY", "aws.s3.amazon.com")
		if err != nil {
			panic(err)
		}
	}
}

func TestMakeConsulPoliciesAndHashiConfigs(t *testing.T) {
	folder := RandString(8)
	defer func() {
		assert.NoError(t, os.RemoveAll(filepath.Clean((filepath.Join(folder)))))
	}()
	inv, err := LoadInventory(filepath.Clean(filepath.Join("testdata", "inventory")))
	assert.NoError(t, err)
	err = makeConsulPolicies(inv, folder)
	assert.NoError(t, err)

	bytes, err := os.ReadFile(filepath.Clean(filepath.Join(folder, "consul", "consul-policies.hcl")))
	assert.NoError(t, err)

	contents := string(bytes)

	assert.Contains(t, contents, `node "ubuntu1"`)

	assert.Equal(t, 16, strings.Count(contents, "node"))

	_, err = os.ReadFile(filepath.Clean(filepath.Join(folder, "consul", "nomad-server-policy.hcl")))
	assert.NoError(t, err)

	_, err = os.ReadFile(filepath.Clean(filepath.Join(folder, "consul", "nomad-client-policy.hcl")))
	assert.NoError(t, err)

	inv, err = LoadInventory(filepath.Clean(filepath.Join("testdata", "inventory")))
	assert.NoError(t, err)
	assert.NoError(t, makeConfigs(*inv, folder, "hetzner"))

	serverBytes, err := os.ReadFile(filepath.Clean(filepath.Join(folder, "consul", "server.j2")))
	assert.NoError(t, err)
	serverConf := string(serverBytes)

	clientBytes, err := os.ReadFile(filepath.Clean(filepath.Join(folder, "consul", "client.j2")))
	assert.NoError(t, err)
	clientConf := string(clientBytes)

	assertFileExists(t, filepath.Join(folder, "nomad", "client.j2"))
	assertFileExists(t, filepath.Join(folder, "nomad", "server.j2"))
	assertFileExists(t, filepath.Join(folder, "nomad", "nomad-server.service"))

	assertFileExists(t, filepath.Join(folder, "nomad", "nomad-client.service"))

	assert.Contains(t, clientConf, `datacenter = "hetzner"`)
	assert.Contains(t, serverConf, `datacenter = "hetzner"`)
	assert.Contains(t, clientConf, `http = 8500`)
	assert.Contains(t, serverConf, `http = 8500`)

	retryJoin := `"10.0.0.3"`
	assert.Contains(t, clientConf, retryJoin)
	assert.Contains(t, serverConf, retryJoin)
	assert.Contains(t, serverConf, `key_file = "/etc/consul.d/certs/hetzner-server-consul-0-key.pem"`)

}

func TestMakeSecrets(t *testing.T) {
	folder := RandString(8)
	defer func() {
		err := os.RemoveAll(folder)
		assert.NoError(t, err)
	}()
	inv, err := LoadInventory(filepath.Join("testdata", "inventory"))
	assert.NoError(t, err)
	err = Secrets(*inv, folder, "dc1")
	assert.NoError(t, err)
	bytes, err := os.ReadFile(filepath.Clean(filepath.Join(folder, "secrets", "secrets.yml")))
	assert.NoError(t, err)
	err = Secrets(*inv, folder, "dc1")
	assert.NoError(t, err)
	bytes2, err := os.ReadFile(filepath.Clean(filepath.Join(folder, "secrets", "secrets.yml")))
	assert.NoError(t, err)
	conf := string(bytes)

	fmt.Println(conf)
	fmt.Println("******")
	fmt.Println(string(bytes2))

	assert.Equal(t, conf, string(bytes2))

	assert.Equal(t, 1, strings.Count(conf, "CONSUL_GOSSIP_KEY"))
	assert.Equal(t, 1, strings.Count(conf, "NOMAD_GOSSIP_KEY"))

	var theMap map[string]interface{}
	err = yaml.Unmarshal([]byte(bytes), &theMap)
	assert.NoError(t, err)
	assert.NotEmpty(t, theMap["CONSUL_GOSSIP_KEY"])

	assert.NotEmpty(t, theMap["NOMAD_GOSSIP_KEY"])

	_, err = os.ReadFile(filepath.Clean(filepath.Join(folder, "secrets", "consul", "consul-agent-ca-key.pem")))
	assert.NoError(t, err)

	_, err = os.ReadFile(filepath.Clean(filepath.Join(folder, "secrets", "consul", "consul-agent-ca.pem")))
	assert.NoError(t, err)
	_, err = os.ReadFile(filepath.Clean(filepath.Join(folder, "secrets", "consul", "dc1-server-consul-0-key.pem")))
	assert.NoError(t, err)
	_, err = os.ReadFile(filepath.Clean(filepath.Join(folder, "secrets", "consul", "dc1-server-consul-0.pem")))
	assert.NoError(t, err)

	nomadDir := filepath.Join(folder, "secrets", "nomad")
	files := []string{
		"cfssl.json",
		"nomad-ca.csr",
		"nomad-ca-key.pem",
		"nomad-ca.pem",
		"cli.csr",
		"cli-key.pem",
		"cli.pem",
		"client.csr",
		"client-key.pem",
		"client.pem",
		"server.csr",
		"server-key.pem",
		"server.pem",
	}

	for _, path := range files {
		assertFileExists(t, filepath.Join(nomadDir, path))
	}

	secrets, err := getSecrets(folder)
	assert.NoError(t, err)
	assert.Equal(t, "TBD", secrets.ConsulBootstrapToken)
	assert.Equal(t, "TBD", secrets.ConsulAgentToken)
	assert.Equal(t, "TBD", secrets.NomadClientConsulToken)
	assert.Equal(t, "TBD", secrets.NomadServerConsulToken)
}

func assertFileExists(t *testing.T, path string) {
	_, err := os.ReadFile(filepath.Clean(path))
	assert.NoError(t, err, path)
}

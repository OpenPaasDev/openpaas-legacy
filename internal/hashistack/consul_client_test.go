package hashistack

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/OpenPaaSDev/openpaas/internal/ansible"
	"github.com/OpenPaaSDev/openpaas/internal/secrets"
	"github.com/stretchr/testify/assert"
)

func TestParseConsulToken(t *testing.T) {

	token, err := parseConsulToken(filepath.Join("testdata", "bootstrap.txt"))
	assert.NoError(t, err)
	assert.Equal(t, "4456269a-e46a-c5bd-08d5-914552161f02", token)

}

func TestGetExports(t *testing.T) {
	exportsExpected := `export CONSUL_HTTP_ADDR="consul-server-1:8501" && export CONSUL_HTTP_TOKEN="foo" && export CONSUL_CLIENT_CERT=testdata/secrets/consul/consul-agent-ca.pem && export CONSUL_CLIENT_KEY=testdata/secrets/consul/consul-agent-ca-key.pem && export CONSUL_HTTP_SSL=true && export CONSUL_HTTP_SSL_VERIFY=false && `
	client := NewConsul(&ansible.Inventory{All: ansible.All{
		Children: ansible.Children{
			ConsulServers: ansible.HostGroup{
				Hosts: map[string]ansible.AnsibleHost{
					"consul-server-1": {PrivateIP: "10.0.0.1", HostName: "consul1"},
				},
			},
		},
	}}, &secrets.Config{
		ConsulBootstrapToken: "foo",
	}, "testdata").(*consulBinary)

	exports, err := client.getExports()
	assert.NoError(t, err)
	fmt.Println(exports)
	assert.Equal(t, exportsExpected, exports)

	client = NewConsul(nil, nil, "").(*consulBinary)

	exports, err = client.getExports()
	assert.NoError(t, err)
	fmt.Println(exports)
	assert.Equal(t, "", exports)

}

func TestRunConsul(t *testing.T) {
	client := NewConsul(nil, nil, "").(*consulBinary)
	err := client.runConsul("version")
	assert.NoError(t, err)

}

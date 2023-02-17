package hashistack

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/OpenPaaSDev/openpaas/internal/ansible"
	"github.com/OpenPaaSDev/openpaas/internal/runtime"
	"github.com/OpenPaaSDev/openpaas/internal/secrets"
)

type Consul interface {
	Bootstrap() (string, error)
	RegisterACL(description, policy string) (string, error)
	UpdateACL(tokenID, policy string) error
	UpdatePolicy(name, file string) error
	RegisterPolicy(name, file string) error
	RegisterIntention(file string) error
	RegisterService(file string) error
}

type consulBinary struct {
	inventory *ansible.Inventory
	secrets   *secrets.Config
	baseDir   string
}

func NewConsul(inventory *ansible.Inventory, secrets *secrets.Config, baseDir string) Consul {
	return &consulBinary{inventory: inventory, secrets: secrets, baseDir: baseDir}
}

func (client *consulBinary) Bootstrap() (string, error) {
	hosts := client.inventory.All.Children.ConsulServers.GetHosts()
	if len(hosts) == 0 {
		return "", fmt.Errorf("no consul servers found in inventory")
	}
	host := hosts[0]
	secretsDir := filepath.Join(client.baseDir, "secrets")

	path := filepath.Join(secretsDir, "consul-bootstrap.token")
	exports := fmt.Sprintf(`export CONSUL_HTTP_ADDR="%s:8501" && export CONSUL_CLIENT_CERT=%s/secrets/consul/consul-agent-ca.pem && export CONSUL_CLIENT_KEY=%s/secrets/consul/consul-agent-ca-key.pem && export CONSUL_HTTP_SSL=true && export CONSUL_HTTP_SSL_VERIFY=false && `, host, client.baseDir, client.baseDir)

	err := runtime.Exec(&runtime.EmptyEnv{}, fmt.Sprintf(`%s consul acl bootstrap > %s`, exports, path), os.Stdout)
	if err != nil {
		return "", err
	}
	token, err := parseConsulToken(path)
	if err != nil {
		return "", err
	}
	client.secrets.ConsulBootstrapToken = token
	return token, nil
}

func (client *consulBinary) RegisterACL(description, policy string) (string, error) {
	tokenPath := filepath.Join(client.baseDir, "secrets", fmt.Sprintf("%s.token", policy))
	err := client.runConsul(fmt.Sprintf(`acl token create -description "%s"  -policy-name %s > %s`, description, policy, tokenPath))
	if err != nil {
		return "", err
	}
	return parseConsulToken(tokenPath)
}

func (client *consulBinary) UpdateACL(tokenID, policy string) error {
	return client.runConsul(fmt.Sprintf(`acl token update -id %s -policy-name=%s`, tokenID, policy))
}

func (client *consulBinary) RegisterPolicy(name, file string) error {
	return client.runConsul(fmt.Sprintf(`acl policy create -name %s -rules @%s`, name, file))
}

func (client *consulBinary) UpdatePolicy(name, file string) error {
	return client.runConsul(fmt.Sprintf(`acl policy update -name %s -rules @%s`, name, file))
}

func (client *consulBinary) RegisterIntention(file string) error {
	return client.runConsul(fmt.Sprintf(`config write %s`, file))
}

func (client *consulBinary) RegisterService(file string) error {
	return client.runConsul(fmt.Sprintf(`services register %s`, file))
}

func (client *consulBinary) getExports() (string, error) {
	if client.inventory == nil && client.secrets == nil {
		return "", nil
	}
	hosts := client.inventory.All.Children.ConsulServers.GetHosts()
	if len(hosts) == 0 {
		return "", fmt.Errorf("no consul servers found in inventory")
	}
	host := hosts[0]

	token := client.secrets.ConsulBootstrapToken
	exports := fmt.Sprintf(`export CONSUL_HTTP_ADDR="%s:8501" && export CONSUL_HTTP_TOKEN="%s" && export CONSUL_CLIENT_CERT=%s/secrets/consul/consul-agent-ca.pem && export CONSUL_CLIENT_KEY=%s/secrets/consul/consul-agent-ca-key.pem && export CONSUL_HTTP_SSL=true && export CONSUL_HTTP_SSL_VERIFY=false && `, host, token, client.baseDir, client.baseDir)
	return exports, nil
}

func (client *consulBinary) runConsul(consulCmd string) error {
	exports, err := client.getExports()
	if err != nil {
		return err
	}
	return runtime.Exec(&runtime.EmptyEnv{}, fmt.Sprintf(`%sconsul %s`, exports, consulCmd), os.Stdout)
}

func parseConsulToken(file string) (string, error) {
	content, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		return "", err
	}

	// Convert []byte to string and print to screen
	text := string(content)
	temp := strings.Split(text, "\n")
	for _, line := range temp {
		if strings.HasPrefix(line, "SecretID:") {
			return strings.ReplaceAll(strings.ReplaceAll(line, "SecretID:", ""), " ", ""), nil
		}
	}
	return "", nil
}

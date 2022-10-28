package hashistack

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/OpenPaas/openpaas/internal/ansible"
	"github.com/OpenPaas/openpaas/internal/secrets"
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

	err := runCmd(fmt.Sprintf(`%s consul acl bootstrap > %s`, exports, path), os.Stdout)
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
	err := client.runConsul(func(exports string) error {
		return runCmd(fmt.Sprintf(`%sconsul acl token create -description "%s"  -policy-name %s > %s`, exports, description, policy, tokenPath), os.Stdout)
	})
	if err != nil {
		return "", err
	}
	return parseConsulToken(tokenPath)
}

func (client *consulBinary) UpdateACL(tokenID, policy string) error {
	return client.runConsul(func(exports string) error {
		return runCmd(fmt.Sprintf(`%sconsul acl token update -id %s -policy-name=%s`, exports, tokenID, policy), os.Stdout)
	})
}

func (client *consulBinary) RegisterPolicy(name, file string) error {
	return client.runConsul(func(exports string) error {
		return runCmd(fmt.Sprintf(`%sconsul acl policy create -name %s -rules @%s`, exports, name, file), os.Stdout)
	})

}

func (client *consulBinary) UpdatePolicy(name, file string) error {
	return client.runConsul(func(exports string) error {
		return runCmd(fmt.Sprintf(`%sconsul acl policy update -name %s -rules @%s`, exports, name, file), os.Stdout)
	})
}

func (client *consulBinary) RegisterIntention(file string) error {
	return client.runConsul(func(exports string) error {
		return runCmd(fmt.Sprintf(`%sconsul config write %s`, exports, file), os.Stdout)
	})
}

func (client *consulBinary) RegisterService(file string) error {
	return client.runConsul(func(exports string) error {
		return runCmd(fmt.Sprintf(`%sconsul services register %s`, exports, file), os.Stdout)
	})
}

func (client *consulBinary) getExports() (string, error) {
	hosts := client.inventory.All.Children.ConsulServers.GetHosts()
	if len(hosts) == 0 {
		return "", fmt.Errorf("no consul servers found in inventory")
	}
	host := hosts[0]

	token := client.secrets.ConsulBootstrapToken
	exports := fmt.Sprintf(`export CONSUL_HTTP_ADDR="%s:8501" && export CONSUL_HTTP_TOKEN="%s" && export CONSUL_CLIENT_CERT=%s/secrets/consul/consul-agent-ca.pem && export CONSUL_CLIENT_KEY=%s/secrets/consul/consul-agent-ca-key.pem && export CONSUL_HTTP_SSL=true && export CONSUL_HTTP_SSL_VERIFY=false && `, host, token, client.baseDir, client.baseDir)
	return exports, nil
}

func (client *consulBinary) runConsul(fn func(string) error) error {
	exports, err := client.getExports()
	if err != nil {
		return err
	}
	return fn(exports)
}

func parseConsulToken(file string) (string, error) {
	content, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		log.Fatal(err)
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

func runCmd(command string, stdOut io.Writer) error {
	cmd := exec.Command("/bin/sh", "-c", command)

	cmd.Stdout = stdOut
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	if err != nil {
		return err
	}
	return cmd.Wait()
}

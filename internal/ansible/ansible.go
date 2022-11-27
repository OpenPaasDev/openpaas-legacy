package ansible

import (
	"fmt"
	"os"

	"github.com/OpenPaas/openpaas/internal/runtime"
)

type Client interface {
	Run(file string) error
}

type ansibleClient struct {
	inventory   string
	secretsFile string
	user        string
	configPath  string
}

func NewClient(inventory, secretsFile, user, configPath string) Client {
	return &ansibleClient{
		inventory:   inventory,
		secretsFile: secretsFile,
		user:        user,
		configPath:  configPath,
	}
}

func (client *ansibleClient) Run(file string) error {
	return runtime.Exec(&runtime.EmptyEnv{}, fmt.Sprintf("ansible-playbook %s -i %s -u %s -e @%s -e @%s", file, client.inventory, client.user, client.secretsFile, client.configPath), os.Stdout)
}

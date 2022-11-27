package internal

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/OpenPaas/openpaas/internal/ansible"
	"github.com/OpenPaas/openpaas/internal/conf"
	sec "github.com/OpenPaas/openpaas/internal/secrets"
)

func GenerateEnvFile(config *conf.Config, targetDir string) error {
	secrets, err := sec.Load(config.BaseDir)
	if err != nil {
		return err
	}
	inv, err := ansible.LoadInventory(filepath.Clean(filepath.Join(config.BaseDir, "inventory")))
	if err != nil {
		return err
	}
	consulServer := inv.All.Children.ConsulServers.GetHosts()[0]
	nomadServer := inv.All.Children.NomadServers.GetHosts()[0]
	vaultServer := inv.All.Children.VaultServers.GetHosts()[0]

	envFile := fmt.Sprintf(`
export CONSUL_HTTP_ADDR=https://%s:8501
export CONSUL_HTTP_TOKEN=%s
export CONSUL_HTTP_SSL=true
export CONSUL_HTTP_SSL_VERIFY=false
export CONSUL_CLIENT_CERT=%s/secrets/consul/consul-agent-ca.pem
export CONSUL_CLIENT_KEY=%s/secrets/consul/consul-agent-ca-key.pem

export VAULT_ADDR=https://%s:8200
export VAULT_SKIP_VERIFY=true
	
export NOMAD_ADDR=https://%s:4646
export NOMAD_CACERT=%s/secrets/nomad/nomad-ca.pem
export NOMAD_CLIENT_CERT=%s/secrets/nomad/client.pem
export NOMAD_CLIENT_KEY=%s/secrets/nomad/client-key.pem	
`, consulServer, secrets.ConsulBootstrapToken, config.BaseDir, config.BaseDir, vaultServer, nomadServer, config.BaseDir, config.BaseDir, config.BaseDir)

	envrcFile := filepath.Join(targetDir, ".envrc")
	bytesRead, err := os.ReadFile(filepath.Clean(envrcFile))
	if err == nil {
		str := string(bytesRead)
		parts := strings.Split(str, "### GENERATED CONFIG BELOW THIS LINE, DO NOT EDIT!")
		if len(parts) != 2 {
			return fmt.Errorf(".envrc file exists, but is not separated by the line\n### GENERATED CONFIG BELOW THIS LINE, DO NOT EDIT! ")
		}
		envFile = fmt.Sprintf("%s\n### GENERATED CONFIG BELOW THIS LINE, DO NOT EDIT!\n%s", parts[0], envFile)
	}
	fmt.Println(envFile)
	return os.WriteFile(filepath.Join(envrcFile), []byte(envFile), 0600)
}

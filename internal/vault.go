package internal

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	_ "embed"

	"github.com/OpenPaas/openpaas/internal/ansible"
	"github.com/OpenPaas/openpaas/internal/secrets"
)

//go:embed templates/vault/nomad-server-policy.hcl
var vaultServerPolicy string

//go:embed templates/vault/token-role.json
var vaultTokenRole string

func generateTLS(config *Config, inventory *ansible.Inventory) error {
	outputDir := filepath.Join(config.BaseDir, "secrets", "vault")
	if _, err := os.Stat(filepath.Join(outputDir, "tls.key")); errors.Is(err, os.ErrNotExist) {
		err := os.MkdirAll(outputDir, 0700)
		if err != nil {
			return err
		}

		hosts := inventory.All.Children.VaultServers.GetPrivateHosts()
		dnsEntries := []string{}
		for _, host := range hosts {
			dnsEntries = append(dnsEntries, fmt.Sprintf("DNS:%s", host))
		}

		keyFile := filepath.Join(outputDir, "tls.key")
		crtFile := filepath.Join(outputDir, "tls.crt")
		dns := strings.Join(dnsEntries, ",")

		cmd := fmt.Sprintf(`openssl req -out %s -new -keyout %s -newkey rsa:4096 -nodes -sha256 -x509 -subj "/O=%s/CN=Vault" -addext "subjectAltName = IP:0.0.0.0,DNS:vault.service.consul,DNS:active.vault.service.consul,%s"`, crtFile, keyFile, config.OrgName, dns)
		fmt.Println(dns)
		fmt.Println(cmd)
		return runCmd("", cmd, os.Stdout)
	}
	return nil
}

func Vault(config *Config, inventory *ansible.Inventory) error {
	outputDir := filepath.Join(config.BaseDir, "secrets", "vault")
	initFile := filepath.Join(outputDir, "init.txt")
	vaultHosts := inventory.All.Children.VaultServers.GetHosts()

	toCopy := map[string]string{
		filepath.Join(config.BaseDir, "vault", "nomad-server-policy.hcl"): vaultServerPolicy,
		filepath.Join(config.BaseDir, "vault", "token-role.json"):         vaultTokenRole,
	}
	for k, v := range toCopy {
		err := os.WriteFile(k, []byte(v), 0600)
		if err != nil {
			return err
		}
	}
	secrets, err := getSecrets(config.BaseDir)
	if err != nil {
		return err
	}
	if _, e := os.Stat(filepath.Clean(initFile)); errors.Is(e, os.ErrNotExist) {

		secrets, err = initVault(config.BaseDir, initFile, vaultHosts, secrets)
		if err != nil {
			return err
		}
	}
	if secrets.VaultConfig.RootToken == "" {
		secrets, err = parseVaultInit(initFile, secrets)
		if err != nil {
			return err
		}
	}

	err = unseal(config.BaseDir, vaultHosts, secrets, false)
	if err != nil {
		return err
	}
	return writeSecrets(config.BaseDir, secrets)
}

func initVault(baseDir, initFile string, vaultHosts []string, secrets *secrets.Config) (*secrets.Config, error) {
	envVars := fmt.Sprintf("export VAULT_SKIP_VERIFY=true && export VAULT_ADDR=https://%s:8200 && ", vaultHosts[0])
	err := runCmd("", fmt.Sprintf("%s vault operator init > %s", envVars, initFile), os.Stdout)
	if err != nil {
		return nil, err
	}
	secrets, err = parseVaultInit(initFile, secrets)
	if err != nil {
		return nil, err
	}
	err = unseal(baseDir, vaultHosts, secrets, true)
	if err != nil {
		return nil, err
	}
	err = enableSecrets(vaultHosts, secrets)
	if err != nil {
		return nil, err
	}

	return secrets, nil
}

func enableSecrets(vaultHosts []string, secrets *secrets.Config) error {
	envVars := fmt.Sprintf("export VAULT_SKIP_VERIFY=true && export VAULT_ADDR=https://%s:8200 && ", vaultHosts[0])
	err := runCmd("", fmt.Sprintf("%s vault login %s && vault secrets enable -path=secret/ kv-v2", envVars, secrets.VaultConfig.RootToken), os.Stdout)
	if err != nil {
		fmt.Println("Error enabling secrets: " + fmt.Sprintf(" %v", err))
	}
	return err
}

func unseal(baseDir string, vaultHosts []string, secrets *secrets.Config, init bool) error {
	for _, host := range vaultHosts {

		var out bytes.Buffer
		envVars := fmt.Sprintf("export VAULT_SKIP_VERIFY=true && export VAULT_ADDR=https://%s:8200 && ", host)
		err := runCmd("", fmt.Sprintf("%s vault status", envVars), &out)
		if err != nil {
			fmt.Println("vault not unsealed")
		}
		if !isVaultUnsealed(out.Bytes()) {
			for _, key := range secrets.VaultConfig.UnsealKeys {
				envVars := fmt.Sprintf("export VAULT_SKIP_VERIFY=true && export VAULT_ADDR=https://%s:8200 && ", host)
				err = runCmd("", fmt.Sprintf("%s vault operator unseal %s", envVars, key), os.Stdout)
				if err != nil {
					return err
				}
			}
		}
	}

	if !init {
		envVars := fmt.Sprintf("export VAULT_SKIP_VERIFY=true && export VAULT_ADDR=https://%s:8200 && ", vaultHosts[0])
		err := runCmd("", fmt.Sprintf("%s vault policy write nomad-server %s/vault/nomad-server-policy.hcl", envVars, baseDir), os.Stdout)
		if err != nil {
			return err
		}
		fmt.Println("gen token")
		err = runCmd("", fmt.Sprintf("%s vault login %s && vault token create -policy nomad-server -period 72h -orphan > %s/secrets/vault/nomad-token.txt", envVars, secrets.VaultConfig.RootToken, baseDir), os.Stdout)
		if err != nil {
			return err
		}
		bytes, err := os.ReadFile(filepath.Clean(filepath.Join(baseDir, "secrets", "vault", "nomad-token.txt")))
		if err != nil {
			return err
		}
		secrets.VaultConfig.NomadRootToken = getVaultToken(bytes)

		fmt.Println("gen token-role")
		return runCmd("", fmt.Sprintf("%s vault login %s && vault write /auth/token/roles/nomad-cluster @%s/vault/token-role.json", envVars, secrets.VaultConfig.RootToken, baseDir), os.Stdout)
	}
	return nil

}

func getVaultToken(bytes []byte) string {
	text := string(bytes)
	temp := strings.Split(text, "\n")
	for _, line := range temp {
		if strings.HasPrefix(line, "token ") {
			return strings.Trim(strings.SplitAfter(line, "token ")[1], " ")
		}
	}
	return ""
}

func isVaultUnsealed(bytes []byte) bool {
	text := string(bytes)
	temp := strings.Split(text, "\n")
	for _, line := range temp {
		if strings.HasPrefix(line, "Sealed") && strings.Contains(line, "true") {
			return false
		}
	}
	return true
}

func parseVaultInit(initFile string, secretConfig *secrets.Config) (*secrets.Config, error) { //nolint
	content, err := os.ReadFile(filepath.Clean(initFile))
	if err != nil {
		log.Fatal(err)
	}

	text := string(content)
	temp := strings.Split(text, "\n")
	secretConfig.VaultConfig = secrets.VaultSecrets{
		UnsealKeys:     []string{},
		RootToken:      "",
		NomadRootToken: "",
	}
	for _, line := range temp {
		if strings.HasPrefix(line, "Unseal Key") {
			secretConfig.VaultConfig.UnsealKeys = append(secretConfig.VaultConfig.UnsealKeys, strings.SplitAfter(line, ": ")[1])
		}
		if strings.HasPrefix(line, "Initial Root Token: ") {
			secretConfig.VaultConfig.RootToken = strings.SplitAfter(line, ": ")[1]
		}
	}
	return secretConfig, nil
}

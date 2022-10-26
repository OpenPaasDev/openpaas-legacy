package internal

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/foomo/htpasswd"
)

func Bootstrap(ctx context.Context, config *Config, configPath string) error {
	inventory := filepath.Join(config.BaseDir, "inventory")
	dcName := config.DC
	user := config.CloudProviderConfig.User
	baseDir := config.BaseDir

	ips, err := GetCloudflareIPs(ctx)
	if err != nil {
		return err
	}
	err = GenerateTerraform(config, ips)
	if err != nil {
		return err
	}

	tf, err := InitTf(ctx, filepath.Join(config.BaseDir, "terraform"), os.Stdout, os.Stderr)
	if err != nil {
		return err
	}
	os.Remove(filepath.Join(config.BaseDir, "inventory-output.json")) //nolint

	err = tf.Apply(ctx, LoadTFExecVars(config))
	if err != nil {
		panic(err)
	}
	f, err := os.OpenFile(filepath.Join(config.BaseDir, "inventory-output.json"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		panic(err)
	}
	defer func() {
		e := f.Close()
		fmt.Println(e)
	}()
	tf, err = InitTf(ctx, filepath.Join(config.BaseDir, "terraform"), f, os.Stderr)
	if err != nil {
		return err
	}
	_, err = tf.Output(ctx)
	if err != nil {
		return err
	}

	err = GenerateInventory(config)
	if err != nil {
		return err
	}
	inv, err := LoadInventory(inventory)
	if err != nil {
		return err
	}
	err = Configure(*inv, baseDir, dcName)
	if err != nil {
		return err
	}
	setup := filepath.Join(baseDir, "base.yml")
	secrets := filepath.Join(baseDir, "secrets", "secrets.yml")
	fmt.Println("sleeping 10s to ensure all nodes are available..")
	time.Sleep(10 * time.Second)

	err = runCmd("", fmt.Sprintf("ansible-playbook %s -i %s -u %s -e @%s -e @%s", setup, inventory, user, secrets, configPath), os.Stdout)
	if err != nil {
		return err
	}
	consulSetup := filepath.Join(baseDir, "consul.yml")
	err = runCmd("", fmt.Sprintf("ansible-playbook %s -i %s -u %s -e @%s -e @%s", consulSetup, inventory, user, secrets, configPath), os.Stdout)
	if err != nil {
		return err
	}

	sec, err := getSecrets(baseDir)
	if err != nil {
		return err
	}
	consul := NewConsul(inv, sec, baseDir)
	hasBootstrapped, err := BootstrapConsul(consul, inv, baseDir)
	if err != nil {
		return err
	}
	if hasBootstrapped {
		fmt.Println("Bootstrapped Consul ACL, re-running Ansible...")
		err = runCmd("", fmt.Sprintf("ansible-playbook %s -i %s -u %s -e @%s -e @%s", consulSetup, inventory, user, secrets, configPath), os.Stdout)
		if err != nil {
			return err
		}
	}
	err = generateTLS(config, inv)
	if err != nil {
		return err
	}
	sec, err = getSecrets(baseDir)
	if err != nil {
		return err
	}
	file := filepath.Join(config.BaseDir, "secrets", "consul.htpasswd")
	name := "consul"
	password := sec.ConsulBootstrapToken
	err = htpasswd.SetPassword(file, name, password, htpasswd.HashBCrypt)
	if err != nil {
		return err
	}
	vaultSetup := filepath.Join(baseDir, "vault.yml")
	err = runCmd("", fmt.Sprintf("ansible-playbook %s -i %s -u %s -e @%s -e @%s", vaultSetup, inventory, user, secrets, configPath), os.Stdout)
	if err != nil {
		return err
	}
	err = Vault(config, inv)
	if err != nil {
		return err
	}

	nomadSetup := filepath.Join(baseDir, "nomad.yml")
	err = runCmd("", fmt.Sprintf("ansible-playbook %s -i %s -u %s -e @%s -e @%s", nomadSetup, inventory, user, secrets, configPath), os.Stdout)
	if err != nil {
		return err
	}

	// 	export NOMAD_ADDR=https://5.75.158.159:4646
	// export NOMAD_CACERT=config/secrets/nomad/nomad-ca.pem
	// export NOMAD_CLIENT_CERT=config/secrets/nomad/client.pem
	// export NOMAD_CLIENT_KEY=config/secrets/nomad/client-key.pem

	return Observability(config, inventory, configPath)
}

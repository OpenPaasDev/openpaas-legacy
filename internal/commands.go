package internal

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/OpenPaaSDev/openpaas/internal/ansible"
	"github.com/OpenPaaSDev/openpaas/internal/conf"
	"github.com/OpenPaaSDev/openpaas/internal/hashistack"
	"github.com/OpenPaaSDev/openpaas/internal/hashistack/vault"
	"github.com/OpenPaaSDev/openpaas/internal/o11y"
	"github.com/OpenPaaSDev/openpaas/internal/util"
	"github.com/foomo/htpasswd"

	secret "github.com/OpenPaaSDev/openpaas/internal/secrets"
)

func Bootstrap(ctx context.Context, config *conf.Config, configPath string) error {
	publicIP, err := util.GetPublicIP(ctx)
	if err != nil {
		return err
	}
	foundIP := false
	for _, ip := range config.CloudProviderConfig.AllowedIPs {
		if ip == fmt.Sprintf("%s/32", publicIP) {
			foundIP = true
			break
		}
	}
	if !foundIP {
		config.CloudProviderConfig.AllowedIPs = append(config.CloudProviderConfig.AllowedIPs, fmt.Sprintf("%s/32", publicIP))
	}
	inventory := filepath.Join(config.BaseDir, "inventory")
	dcName := config.DC
	user := config.CloudProviderConfig.User
	baseDir := config.BaseDir

	err = hashistack.GenerateTerraform(config)
	if err != nil {
		return err
	}

	tf, err := hashistack.InitTf(ctx, filepath.Join(config.BaseDir, "terraform"), os.Stdout, os.Stderr)
	if err != nil {
		return err
	}
	os.Remove(filepath.Join(config.BaseDir, "inventory-output.json")) //nolint

	err = tf.Apply(ctx, conf.LoadTFExecVars(config))
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
	tf, err = hashistack.InitTf(ctx, filepath.Join(config.BaseDir, "terraform"), f, os.Stderr)
	if err != nil {
		return err
	}
	_, err = tf.Output(ctx)
	if err != nil {
		return err
	}

	inv, err := ansible.GenerateInventory(config)
	if err != nil {
		return err
	}

	err = Configure(inv, baseDir, dcName)
	if err != nil {
		return err
	}

	secrets := secret.File(baseDir)

	ansibleClient := ansible.NewClient(inventory, secrets, user, configPath)

	err = ansibleClient.Run(filepath.Join(baseDir, "base.yml"))
	if err != nil {
		return err
	}
	consulSetup := filepath.Join(baseDir, "consul.yml")
	err = ansibleClient.Run(consulSetup)
	if err != nil {
		return err
	}

	sec, err := secret.Load(baseDir)
	if err != nil {
		return err
	}

	consul := hashistack.NewConsul(inv, sec, baseDir)
	hasBootstrapped, err := BootstrapConsul(consul, inv, sec, baseDir)
	if err != nil {
		return err
	}
	if hasBootstrapped {
		fmt.Println("Bootstrapped Consul ACL, re-running Ansible...")
		err = ansibleClient.Run(consulSetup)
		if err != nil {
			return err
		}
	}

	err = vault.GenerateTLS(config, inv)
	if err != nil {
		return err
	}

	err = htpasswd.SetPassword(filepath.Join(config.BaseDir, "secrets", "consul.htpasswd"),
		"consul", sec.ConsulBootstrapToken, htpasswd.HashBCrypt)
	if err != nil {
		return err
	}

	err = ansibleClient.Run(filepath.Join(baseDir, "vault.yml"))
	if err != nil {
		return err
	}
	err = vault.Init(config, inv, sec)
	if err != nil {
		return err
	}

	err = ansibleClient.Run(filepath.Join(baseDir, "nomad.yml"))
	if err != nil {
		return err
	}

	nomadSecretDir := filepath.Join(baseDir, "secrets", "nomad")

	nomadClient := hashistack.NewNomadClient("",
		fmt.Sprintf("https://%s:4646", inv.All.Children.NomadServers.GetHosts()[0]),
		filepath.Join(nomadSecretDir, "nomad-ca.pem"),
		filepath.Join(nomadSecretDir, "client.pem"),
		filepath.Join(nomadSecretDir, "client-key.pem"),
	)

	err = nomadClient.RunJob(filepath.Join(baseDir, "nomad", "web.hcl"))
	if err != nil {
		return err
	}

	return o11y.Init(config, inventory, configPath, sec, consul, ansibleClient)
}

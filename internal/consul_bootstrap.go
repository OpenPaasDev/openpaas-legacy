package internal

import (
	"fmt"
	"path/filepath"

	"github.com/OpenPaaSDev/openpaas/internal/ansible"
	"github.com/OpenPaaSDev/openpaas/internal/hashistack"
	"github.com/OpenPaaSDev/openpaas/internal/secrets"
)

func regenerateConsulPolicies(consul hashistack.Consul, inventory *ansible.Inventory, baseDir string) error {
	err := makeConsulPolicies(inventory, baseDir)
	if err != nil {
		return err
	}
	fmt.Println("Updating consul policies")
	policyConsul := filepath.Join(baseDir, "consul", "consul-policies.hcl")

	return consul.UpdatePolicy("consul-policies", policyConsul)
}

func BootstrapConsul(consul hashistack.Consul, inventory *ansible.Inventory, sec *secrets.Config, baseDir string) (bool, error) {

	if sec.ConsulBootstrapToken != "TBD" {
		err := regenerateConsulPolicies(consul, inventory, baseDir)
		return false, err
	}
	token, err := consul.Bootstrap()
	if err != nil {
		return false, err
	}

	sec.ConsulBootstrapToken = token

	policies := map[string]string{
		"consul-policies":    filepath.Join(baseDir, "consul", "consul-policies.hcl"),
		"nomad-client":       filepath.Join(baseDir, "consul", "nomad-client-policy.hcl"),
		"fabio":              filepath.Join(baseDir, "consul", "fabio-policy.hcl"),
		"nomad-server":       filepath.Join(baseDir, "consul", "nomad-server-policy.hcl"),
		"prometheus":         filepath.Join(baseDir, "consul", "prometheus-policy.hcl"),
		"anonymous-dns-read": filepath.Join(baseDir, "consul", "anonymous-policy.hcl"),
		"vault":              filepath.Join(baseDir, "consul", "vault-policy.hcl"),
	}

	for k, v := range policies {
		err = consul.RegisterPolicy(k, v)
		if err != nil {
			return false, err
		}
	}

	err = consul.UpdateACL("anonymous", "anonymous-dns-read")
	if err != nil {
		return false, err
	}

	acls := map[string]string{
		"agent token":        "consul-policies",
		"client token":       "nomad-client",
		"nomad server token": "nomad-server",
		"prometheus token":   "prometheus",
		"vault token":        "vault",
		"fabio token":        "fabio",
	}
	tokens := map[string]string{}

	for k, v := range acls {
		clientToken, e := consul.RegisterACL(k, v)
		if e != nil {
			return false, e
		}
		tokens[v] = clientToken
	}

	sec.ConsulAgentToken = tokens["consul-policies"]
	sec.NomadClientConsulToken = tokens["nomad-client"]
	sec.NomadServerConsulToken = tokens["nomad-server"]
	sec.PrometheusConsulToken = tokens["prometheus"]
	sec.FabioConsulToken = tokens["fabio"]
	sec.VaultConsulToken = tokens["vault"]

	err = sec.Write(baseDir)

	return true, err
}

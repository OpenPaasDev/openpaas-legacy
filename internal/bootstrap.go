package internal

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/OpenPaaSDev/openpaas/internal/ansible"
	"github.com/OpenPaaSDev/openpaas/internal/runtime"
	sec "github.com/OpenPaaSDev/openpaas/internal/secrets"
)

//go:embed templates/consul/resolved.conf
var resolvedConf string

//go:embed templates/consul/anonymous-policy.hcl
var anonymousDns string

//go:embed templates/consul/install-exporter.sh
var installExporter string

//go:embed templates/consul/consul-exporter.service
var consulExporterService string

//go:embed templates/consul/consul-policies.hcl
var consulPolicies string

//go:embed templates/consul/fabio-policy.hcl
var fabioPolicy string

//go:embed templates/fabio/fabio.service
var fabioService string

//go:embed templates/fabio/install-fabio.sh
var fabioInstaller string

//go:embed templates/fabio/fabio.j2
var fabioConf string

//go:embed templates/consul/nomad-client-policy.hcl
var nomadClientPolicy string

//go:embed templates/consul/nomad-server-policy.hcl
var nomadServerPolicy string

//go:embed templates/consul/vault-policy.hcl
var vaultPolicy string

//go:embed templates/consul/prometheus-policy.hcl
var prometheusPolicy string

//go:embed templates/consul/consul-server-config.hcl
var consulServer string

//go:embed templates/consul/consul-client-config.hcl
var consulClient string

//go:embed templates/nomad/cfssl.json
var cfssl string

//go:embed templates/nomad/client.j2
var nomadClient string

//go:embed templates/nomad/server.j2
var nomadServer string

//go:embed templates/nomad/nomad.service
var nomadService string

//go:embed templates/nomad/web.hcl
var nomadHealthCheck string

//go:embed templates/consul/consul.service
var consulService string

//go:embed templates/vault/vault.service
var vaultService string

//go:embed templates/vault/config.hcl
var vaultConf string

//go:embed templates/ansible/base.yml
var baseAnsible string

//go:embed templates/ansible/consul.yml
var consulAnsible string

//go:embed templates/ansible/nomad.yml
var nomadAnsible string

//go:embed templates/ansible/vault.yml
var vaultAnsible string

// calculate bootstrap expect from files
func Configure(inventory *ansible.Inventory, baseDir, dcName string) error {

	err := os.MkdirAll(filepath.Join(baseDir), 0750)
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(baseDir, "base.yml"), []byte(strings.ReplaceAll(baseAnsible, "dc1", dcName)), 0600)
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(baseDir, "consul.yml"), []byte(strings.ReplaceAll(consulAnsible, "dc1", dcName)), 0600)
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(baseDir, "nomad.yml"), []byte(strings.ReplaceAll(nomadAnsible, "dc1", dcName)), 0600)
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(baseDir, "vault.yml"), []byte(strings.ReplaceAll(vaultAnsible, "dc1", dcName)), 0600)
	if err != nil {
		return err
	}

	err = makeConsulPolicies(inventory, baseDir)
	if err != nil {
		return err
	}
	err = makeConfigs(inventory, baseDir, dcName)
	if err != nil {
		return err
	}

	err = Secrets(inventory, baseDir, dcName)
	return err
}

func makeConsulPolicies(inventory *ansible.Inventory, baseDir string) error {

	err := os.MkdirAll(filepath.Join(baseDir, "consul"), 0750)
	if err != nil {
		return err
	}
	err = os.MkdirAll(filepath.Join(baseDir, "fabio"), 0750)
	if err != nil {
		return err
	}
	_ = os.Remove(filepath.Join(baseDir, "consul", "consul-policies.hcl"))

	hosts := inventory.GetAllPrivateHosts()

	tmpl, e := template.New("consul-policies").Parse(consulPolicies)
	if e != nil {
		return e
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, map[string][]string{"Hosts": hosts})
	if err != nil {
		return err
	}

	output := buf.Bytes()
	err = os.WriteFile(filepath.Join(baseDir, "consul", "consul-policies.hcl"), output, 0600)
	if err != nil {
		return err
	}

	toCopy := map[string]string{
		filepath.Join(baseDir, "consul", "fabio-policy.hcl"):        fabioPolicy,
		filepath.Join(baseDir, "consul", "nomad-client-policy.hcl"): nomadClientPolicy,
		filepath.Join(baseDir, "consul", "install-exporter.sh"):     installExporter,
		filepath.Join(baseDir, "consul", "consul-exporter.service"): consulExporterService,
		filepath.Join(baseDir, "consul", "nomad-server-policy.hcl"): nomadServerPolicy,
		filepath.Join(baseDir, "consul", "prometheus-policy.hcl"):   prometheusPolicy,
		filepath.Join(baseDir, "consul", "anonymous-policy.hcl"):    anonymousDns,
		filepath.Join(baseDir, "consul", "vault-policy.hcl"):        vaultPolicy,
		filepath.Join(baseDir, "fabio", "fabio.j2"):                 fabioConf,
		filepath.Join(baseDir, "fabio", "fabio.service"):            fabioService,
		filepath.Join(baseDir, "fabio", "install-fabio.sh"):         fabioInstaller,
	}
	for k, v := range toCopy {
		err = os.WriteFile(k, []byte(v), 0600)
		if err != nil {
			return err
		}
	}
	return nil
}

func makeConfigs(inventory *ansible.Inventory, baseDir, dcName string) error {
	hostMap := make(map[string]string)
	hosts := ""
	first := true

	for _, v := range inventory.All.Children.ConsulServers.Hosts {
		if first {
			hosts = fmt.Sprintf(`"%v"`, v.PrivateIP)
			first = false
		} else {
			hosts = hosts + `, ` + fmt.Sprintf(`"%v"`, v.PrivateIP)
		}
		hostMap[v.PrivateIP] = v.PrivateIP
	}
	clientWithDC := strings.ReplaceAll(consulClient, "dc1", dcName)
	clientWithDC = strings.ReplaceAll(clientWithDC, "join_servers", hosts)
	err := os.WriteFile(filepath.Join(baseDir, "consul", "client.j2"), []byte(clientWithDC), 0600)
	if err != nil {
		return err
	}

	serverWithDC := strings.ReplaceAll(consulServer, "dc1", dcName)
	serverWithDC = strings.ReplaceAll(serverWithDC, "join_servers", hosts)
	serverWithDC = strings.ReplaceAll(serverWithDC, "EXPECTS_NO", fmt.Sprintf("%v", len(inventory.All.Children.ConsulServers.GetHosts())))
	err = os.WriteFile(filepath.Join(baseDir, "consul", "server.j2"), []byte(serverWithDC), 0600)
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(baseDir, "consul", "resolved.conf"), []byte(resolvedConf), 0600)
	if err != nil {
		return err
	}
	err = os.MkdirAll(filepath.Join(baseDir, "nomad"), 0750)
	if err != nil {
		return err
	}
	err = os.MkdirAll(filepath.Join(baseDir, "vault"), 0750)
	if err != nil {
		return err
	}
	nomadServerService := strings.ReplaceAll(nomadService, "nomad_user", "nomad")
	nomadClientService := strings.ReplaceAll(nomadService, "nomad_user", "root")

	nomadServer = strings.ReplaceAll(nomadServer, "EXPECTS_NO", fmt.Sprintf("%v", len(inventory.All.Children.NomadServers.GetHosts())))
	nomadServer = strings.ReplaceAll(nomadServer, "dc1", dcName)
	nomadClient = strings.ReplaceAll(nomadClient, "dc1", dcName)

	toWrite := map[string]string{
		filepath.Join(baseDir, "nomad", "server.j2"):            nomadServer,
		filepath.Join(baseDir, "nomad", "client.j2"):            nomadClient,
		filepath.Join(baseDir, "nomad", "nomad-server.service"): nomadServerService,
		filepath.Join(baseDir, "nomad", "nomad-client.service"): nomadClientService,

		filepath.Join(baseDir, "nomad", "web.hcl"):         strings.Replace(nomadHealthCheck, "{DATACENTRE}", dcName, -1),
		filepath.Join(baseDir, "consul", "consul.service"): consulService,
		filepath.Join(baseDir, "vault", "vault.service"):   vaultService,
		filepath.Join(baseDir, "vault", "config.hcl"):      vaultConf,
	}

	for k, v := range toWrite {
		err = os.WriteFile(k, []byte(v), 0600)
		if err != nil {
			return err
		}
	}

	return nil
}

func Secrets(inventory *ansible.Inventory, baseDir, dcName string) error {
	var out bytes.Buffer
	err := runtime.Exec(&runtime.EmptyEnv{}, "consul keygen", &out)
	if err != nil {
		return err
	}
	consulSecretDir := filepath.Join(baseDir, "secrets", "consul")
	nomadSecretDir := filepath.Join(baseDir, "secrets", "nomad")
	err = os.MkdirAll(consulSecretDir, 0750)
	if err != nil {
		return err
	}
	consulGossipKey := strings.ReplaceAll(out.String(), "\n", "")

	var out2 bytes.Buffer
	err = runtime.Exec(&runtime.EmptyEnv{}, "nomad operator keygen", &out2)

	if err != nil {
		return err
	}
	nomadGossipKey := strings.ReplaceAll(out2.String(), "\n", "")
	if os.Getenv("S3_ENDPOINT") == "" || os.Getenv("S3_SECRET_KEY") == "" || os.Getenv("S3_ACCESS_KEY") == "" {
		return fmt.Errorf("s3 compatible env variables missing for storing state: please set S3_ENDPOINT, S3_SECRET_KEY & S3_ACCESS_KEY")
	}

	secrets := &sec.Config{
		ConsulGossipKey:        consulGossipKey,
		NomadGossipKey:         nomadGossipKey,
		NomadClientConsulToken: "TBD",
		NomadServerConsulToken: "TBD",
		ConsulAgentToken:       "TBD",
		ConsulBootstrapToken:   "TBD",
		FabioConsulToken:       "TBD",
		S3Endpoint:             os.Getenv("S3_ENDPOINT"),
		S3SecretKey:            os.Getenv("S3_SECRET_KEY"),
		S3AccessKey:            os.Getenv("S3_ACCESS_KEY"),
	}

	if _, err1 := os.Stat(sec.File(baseDir)); errors.Is(err1, os.ErrNotExist) {
		e := secrets.Write(baseDir)
		if e != nil {
			return e
		}
	}

	if err != nil {
		return err
	}
	err = os.MkdirAll(nomadSecretDir, 0750)
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(baseDir, "secrets", "consul", "consul-agent-ca.pem")); errors.Is(err, os.ErrNotExist) {
		err = runtime.Exec(runtime.EnvWithDir(consulSecretDir), "consul tls ca create", os.Stdout)
		if err != nil {
			return err
		}
		err = runtime.Exec(runtime.EnvWithDir(consulSecretDir), fmt.Sprintf("consul tls cert create -server -dc %s", dcName), os.Stdout)
		if err != nil {
			return err
		}

	}

	if _, err := os.Stat(filepath.Join(baseDir, "secrets", "nomad", "cli.pem")); errors.Is(err, os.ErrNotExist) {
		err = runtime.Exec(runtime.EnvWithDir(nomadSecretDir), "cfssl print-defaults csr | cfssl gencert -initca - | cfssljson -bare nomad-ca", os.Stdout)
		if err != nil {
			return err
		}
		hosts := inventory.All.Children.NomadServers.GetHosts()
		privateHosts := inventory.All.Children.NomadServers.GetPrivateHosts()
		hostString := fmt.Sprintf("server.global.nomad,%s,%s", strings.Join(hosts, ","), strings.Join(privateHosts, ","))
		fmt.Println("generating cert for hosts: " + hostString)

		err = os.WriteFile(filepath.Join(nomadSecretDir, "cfssl.json"), []byte(cfssl), 0600)
		if err != nil {
			return err
		}
		err = runtime.Exec(runtime.EnvWithDir(nomadSecretDir), fmt.Sprintf(`echo '{}' | cfssl gencert -ca=nomad-ca.pem -ca-key=nomad-ca-key.pem -config=cfssl.json -hostname="%s" - | cfssljson -bare server`, hostString), os.Stdout)
		if err != nil {
			return err
		}

		err = runtime.Exec(runtime.EnvWithDir(nomadSecretDir), fmt.Sprintf(`echo '{}' | cfssl gencert -ca=nomad-ca.pem -ca-key=nomad-ca-key.pem -config=cfssl.json -hostname="%s" - | cfssljson -bare client`, hostString), os.Stdout)
		if err != nil {
			return err
		}

		err = runtime.Exec(runtime.EnvWithDir(nomadSecretDir), fmt.Sprintf(`echo '{}' | cfssl gencert -ca=nomad-ca.pem -ca-key=nomad-ca-key.pem -config=cfssl.json -hostname="%s" - | cfssljson -bare cli`, hostString), os.Stdout)
		if err != nil {
			return err
		}

	}
	return nil
}

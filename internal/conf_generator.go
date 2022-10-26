package internal

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

//go:embed templates/terraform/hetzner/main.tf
var hetznerMain string

//go:embed templates/terraform/hetzner/vars.tf
var hetznerVars string

func GenerateTerraform(config *Config, ips *CloudflareIPs) error {
	settings := map[string]struct {
		Main string
		Vars string
	}{
		"hetzner": {
			Main: hetznerMain,
			Vars: hetznerVars,
		},
	}

	tfSettings, ok := settings[config.CloudProviderConfig.Provider]
	if !ok {
		return fmt.Errorf("%s is not a supported cloud provider", config.CloudProviderConfig.Provider)
	}

	tmpl, e := template.New("tf-vars").Parse(tfSettings.Vars)
	if e != nil {
		return e
	}
	var buf bytes.Buffer

	allowedIps := []string{}

	allowedIps = append(allowedIps, ips.IPV4...)
	allowedIps = append(allowedIps, ips.IPV6...)

	config.CloudProviderConfig.ProviderSettings["https_allowed_ips"] = allowedIps

	err := tmpl.Execute(&buf, config)
	if err != nil {
		return err
	}
	err = os.MkdirAll(filepath.Clean(filepath.Join(config.BaseDir, "terraform")), 0750)
	if err != nil {
		return err
	}
	folder := filepath.Join(config.BaseDir, "terraform")

	err = os.WriteFile(filepath.Clean(filepath.Join(folder, "vars.tf")), buf.Bytes(), 0600)
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Clean(filepath.Join(folder, "main.tf")), []byte(hetznerMain), 0600)
	if err != nil {
		return err
	}
	return nil
}

func GenerateEnvFile(config *Config, targetDir string) error {
	secrets, err := getSecrets(config.BaseDir)
	if err != nil {
		return err
	}
	inv, err := LoadInventory(filepath.Clean(filepath.Join(config.BaseDir, "inventory")))
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

func GenerateInventory(config *Config) error {
	jsonFile, err := os.Open(filepath.Clean(filepath.Join(config.BaseDir, "inventory-output.json")))
	if err != nil {
		return err
	}
	defer func() {
		e := jsonFile.Close()
		fmt.Println(e)
	}()
	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		return err
	}
	var inventory InventoryJson

	err = json.Unmarshal(byteValue, &inventory)
	if err != nil {
		return err
	}

	inv := Inventory{
		All: All{
			Children: Children{
				Clients:       HostGroup{Hosts: make(map[string]AnsibleHost)},
				NomadServers:  HostGroup{Hosts: make(map[string]AnsibleHost)},
				ConsulServers: HostGroup{Hosts: make(map[string]AnsibleHost)},
				VaultServers:  HostGroup{Hosts: make(map[string]AnsibleHost)},
				Grafana:       HostGroup{Hosts: make(map[string]AnsibleHost)},
				Prometheus:    HostGroup{Hosts: make(map[string]AnsibleHost)},
				Loki:          HostGroup{Hosts: make(map[string]AnsibleHost)},
				Tempo:         HostGroup{Hosts: make(map[string]AnsibleHost)},
			},
		},
	}

	consulHosts := inventory.ConsulServers.Value
	if len(consulHosts) == 0 {
		consulHosts = inventory.NomadServers.Value
	}

	for _, v := range consulHosts {
		fmt.Println(v)
		found := false
		for _, vol := range inventory.ConsulVolumes.Value {
			if fmt.Sprintf("%v", vol.ServerID) == v.ServerID {
				found = true
				inv.All.Children.ConsulServers.Hosts[v.Host] = AnsibleHost{
					PrivateIP: v.PrivateIP,
					HostName:  v.HostName,
					Mounts: []Mount{
						{
							Name:      "consul",
							Path:      "/opt/consul",
							MountPath: vol.Mount,
							IsNomad:   false,
							Owner:     "consul",
						},
					},
					ExtraVars: map[string]string{
						"bootstrap_expect": fmt.Sprintf("%v", len(consulHosts)),
						"datacenter":       config.DC,
					},
				}
			}
		}
		if !found {
			inv.All.Children.ConsulServers.Hosts[v.Host] = AnsibleHost{
				PrivateIP: v.PrivateIP,
				HostName:  v.HostName,
			}
		}
	}

	for _, v := range inventory.NomadServers.Value {
		inv.All.Children.NomadServers.Hosts[v.Host] = AnsibleHost{
			PrivateIP: v.PrivateIP,
			HostName:  v.HostName,
			ExtraVars: map[string]string{
				"bootstrap_expect": fmt.Sprintf("%v", len(inventory.NomadServers.Value)),
				"datacenter":       config.DC,
			},
		}
	}

	for _, v := range inventory.VaultServers.Value {
		inv.All.Children.VaultServers.Hosts[v.Host] = AnsibleHost{
			PrivateIP: v.PrivateIP,
			HostName:  v.HostName,
		}
	}

	for _, v := range inventory.Clients.Value {
		mounts := []Mount{}
		for _, vol := range inventory.ClientVolumes.Value {
			if fmt.Sprintf("%v", vol.ServerID) == v.ServerID {
				mounts = append(mounts, Mount{
					Name:      vol.Name,
					Path:      vol.Path,
					MountPath: vol.Mount,
					IsNomad:   true,
					Owner:     config.CloudProviderConfig.User,
				})
			}
		}
		inv.All.Children.Clients.Hosts[v.Host] = AnsibleHost{
			PrivateIP: v.PrivateIP,
			HostName:  v.HostName,
			Mounts:    mounts,
		}

	}

	if len(inventory.ObservabilityServers.Value) == 1 {
		v := inventory.ObservabilityServers.Value[0]
		inv.All.Children.Grafana.Hosts[v.Host] = AnsibleHost{
			PrivateIP: v.PrivateIP,
			HostName:  v.HostName,
		}
		inv.All.Children.Prometheus.Hosts[v.Host] = AnsibleHost{
			PrivateIP: v.PrivateIP,
			HostName:  v.HostName,
		}
		inv.All.Children.Loki.Hosts[v.Host] = AnsibleHost{
			PrivateIP: v.PrivateIP,
			HostName:  v.HostName,
		}
		inv.All.Children.Tempo.Hosts[v.Host] = AnsibleHost{
			PrivateIP: v.PrivateIP,
			HostName:  v.HostName,
		}
	} else {
		v := inventory.ObservabilityServers.Value[0]
		inv.All.Children.Grafana.Hosts[v.Host] = AnsibleHost{
			PrivateIP: v.PrivateIP,
			HostName:  v.HostName,
		}
		v = inventory.ObservabilityServers.Value[1]
		inv.All.Children.Prometheus.Hosts[v.Host] = AnsibleHost{
			PrivateIP: v.PrivateIP,
			HostName:  v.HostName,
		}
		v = inventory.ObservabilityServers.Value[2]
		inv.All.Children.Loki.Hosts[v.Host] = AnsibleHost{
			PrivateIP: v.PrivateIP,
			HostName:  v.HostName,
		}
		v = inventory.ObservabilityServers.Value[3]
		inv.All.Children.Tempo.Hosts[v.Host] = AnsibleHost{
			PrivateIP: v.PrivateIP,
			HostName:  v.HostName,
		}
	}

	bytes, err := yaml.Marshal(inv)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Clean(filepath.Join(config.BaseDir, "inventory")), bytes, 0600)
}

type InventoryJson struct {
	Clients              HostValues `json:"client_servers"`
	NomadServers         HostValues `json:"nomad_servers"`
	ObservabilityServers HostValues `json:"o11y_servers"`
	VaultServers         HostValues `json:"vault_servers"`
	ConsulServers        HostValues `json:"consul_servers"`
	ConsulVolumes        Volumes    `json:"consul_volumes"`
	ClientVolumes        Volumes    `json:"client_volumes"`
}
type HostValues struct {
	Value []Host `json:"value"`
}

type Host struct {
	Host      string `json:"host"`
	HostName  string `json:"host_name"`
	PrivateIP string `json:"private_ip"`
	ServerID  string `json:"server_id"`
}
type Volumes struct {
	Value []Volume `json:"value"`
}

type Volume struct {
	Mount    string `json:"mount"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	ServerID int    `json:"server_id"`
}

type Inventory struct {
	All All `yaml:"all"`
}

type All struct {
	Children Children `yaml:"children"`
}

type Children struct {
	Clients       HostGroup `yaml:"clients"`
	NomadServers  HostGroup `yaml:"nomad_servers"`
	VaultServers  HostGroup `yaml:"vault_servers"`
	ConsulServers HostGroup `yaml:"consul_servers"`
	Prometheus    HostGroup `yaml:"prometheus"`
	Grafana       HostGroup `yaml:"grafana"`
	Loki          HostGroup `yaml:"loki"`
	Tempo         HostGroup `yaml:"tempo"`
}

type HostGroup struct {
	Hosts map[string]AnsibleHost `yaml:"hosts"`
}

type AnsibleHost struct {
	PrivateIP string            `yaml:"private_ip"`
	HostName  string            `yaml:"host_name"`
	Mounts    []Mount           `yaml:"mounts"`
	ExtraVars map[string]string `yaml:"extra_vars"`
}

type Mount struct {
	Name      string `yaml:"name"`
	Path      string `yaml:"path"`
	MountPath string `yaml:"mount_path"`
	IsNomad   bool   `yaml:"is_nomad"`
	Owner     string `yaml:"owner"`
}

func (group *HostGroup) GetHosts() []string {
	res := []string{}
	for k := range group.Hosts {
		res = append(res, k)
	}
	return res
}

func (group *HostGroup) GetPrivateHosts() []string {
	res := []string{}
	for _, v := range group.Hosts {
		res = append(res, v.PrivateIP)
	}
	return res
}

func (group *HostGroup) GetPrivateHostNames() []string {
	res := []string{}
	for _, v := range group.Hosts {
		res = append(res, v.HostName)
	}
	return res
}

func (inv *Inventory) GetAllPrivateHosts() []string {
	hosts := []string{}
	rawHosts := []HostGroup{}
	seenHosts := make(map[string]string)

	rawHosts = append(rawHosts, inv.All.Children.Clients)
	rawHosts = append(rawHosts, inv.All.Children.ConsulServers)
	rawHosts = append(rawHosts, inv.All.Children.NomadServers)
	rawHosts = append(rawHosts, inv.All.Children.VaultServers)
	rawHosts = append(rawHosts, inv.All.Children.Prometheus)
	rawHosts = append(rawHosts, inv.All.Children.Grafana)
	rawHosts = append(rawHosts, inv.All.Children.Loki)
	rawHosts = append(rawHosts, inv.All.Children.Tempo)

	for _, hostGroup := range rawHosts {
		for _, host := range hostGroup.GetPrivateHosts() {
			if _, ok := seenHosts[host]; !ok {
				hosts = append(hosts, host)
				seenHosts[host] = host
			}
		}
		for _, host := range hostGroup.GetPrivateHostNames() {
			if _, ok := seenHosts[host]; !ok {
				hosts = append(hosts, host)
				seenHosts[host] = host
			}
		}
	}

	return hosts
}

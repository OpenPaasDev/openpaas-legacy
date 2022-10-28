package ansible

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

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

func LoadInventory(file string) (*Inventory, error) {
	bytes, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		return nil, err
	}
	var config Inventory
	err = yaml.Unmarshal(bytes, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

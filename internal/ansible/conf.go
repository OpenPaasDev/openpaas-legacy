package ansible

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/OpenPaaSDev/openpaas/internal/conf"
	"gopkg.in/yaml.v3"
)

type Inventory struct {
	All All `yaml:"all"`
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
type Volumes struct {
	Value []Volume `json:"value"`
}

type Volume struct {
	Mount    string `json:"mount"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	ServerID int    `json:"server_id"`
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

func GenerateInventory(config *conf.Config) (*Inventory, error) {
	jsonFile, err := os.Open(filepath.Clean(filepath.Join(config.BaseDir, "inventory-output.json")))
	if err != nil {
		return nil, err
	}
	defer func() {
		e := jsonFile.Close()
		fmt.Println(e)
	}()
	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		return nil, err
	}
	var inventory InventoryJson

	err = json.Unmarshal(byteValue, &inventory)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	err = os.WriteFile(filepath.Clean(filepath.Join(config.BaseDir, "inventory")), bytes, 0600)
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

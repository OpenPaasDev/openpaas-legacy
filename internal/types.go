package internal

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform-exec/tfexec"
	"gopkg.in/yaml.v3"
)

type Config struct {
	DC                  string              `yaml:"dc_name"`
	BaseDir             string              `yaml:"base_dir"`
	OrgName             string              `yaml:"org_name"`
	CloudProviderConfig CloudProvider       `yaml:"cloud_provider_config"`
	ClusterConfig       ClusterConfig       `yaml:"cluster_config"`
	ObservabilityConfig ObservabilityConfig `yaml:"observability_config"`
}

type ClusterConfig struct {
	Servers               int            `yaml:"servers"`
	Clients               int            `yaml:"clients"`
	ConsulVolumeSize      int            `yaml:"consul_volume_size"`
	VaultServers          int            `yaml:"vault_servers"`
	SeparateConsulServers bool           `yaml:"separate_consul_servers"`
	ClientVolumes         []ClientVolume `yaml:"client_volumes"`
	Ingress               IngressConfig  `yaml:"ingress"`
}

type IngressConfig struct {
	ManagementDomain string `yaml:"management_domain"`
}

type ClientVolume struct {
	Name   string `yaml:"name"`
	Client string `yaml:"client"`
	Path   string `yaml:"path"`
	Size   int    `yaml:"size"`
}

type CloudProvider struct {
	User             string                 `yaml:"sudo_user"`
	Dir              string                 `yaml:"sudo_dir"`
	NetworkInterface string                 `yaml:"internal_network_interface_name"`
	Provider         string                 `yaml:"provider"`
	ProviderSettings map[string]interface{} `yaml:"provider_settings"`
}

type ObservabilityConfig struct {
	TempoBucket   string `yaml:"tempo_bucket"`
	LokiBucket    string `yaml:"loki_bucket"`
	MultiInstance bool   `yaml:"multi_instance"`
}

type HetznerResourceNames struct {
	BaseServerName string `yaml:"base_server_name"`
	FirewallName   string `yaml:"firewall_name"`
	NetworkName    string `yaml:"network_name"`
}

type HetznerSettings struct {
	Location                  string               `yaml:"location"`
	SSHKeys                   []string             `yaml:"ssh_keys"`
	AllowedIPs                []string             `yaml:"allowed_ips"`
	ServerInstanceType        string               `yaml:"server_instance_type"`
	ClientInstanceType        string               `yaml:"client_instance_type"`
	ObservabilityInstanceType string               `yaml:"observability_instance_type"`
	ResourceNames             HetznerResourceNames `yaml:"resource_names"`
}

type TFVarsConfig struct {
	ClusterConfig  ClusterConfig
	ProviderConfig interface{}
}

func LoadConfig(file string) (*Config, error) {
	bytes, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		return nil, err
	}
	var config Config
	err = yaml.Unmarshal(bytes, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func LoadTFVarsConfig(config Config) (*TFVarsConfig, error) {
	var providerConfig interface{}
	if config.CloudProviderConfig.Provider == "hetzner" {
		var hetznerConfig HetznerSettings
		bytes, err := yaml.Marshal(config.CloudProviderConfig.ProviderSettings)
		if err != nil {
			return nil, err
		}

		err = yaml.Unmarshal(bytes, &hetznerConfig)
		if err != nil {
			return nil, err
		}
		providerConfig = hetznerConfig
	}

	return &TFVarsConfig{
		ClusterConfig:  config.ClusterConfig,
		ProviderConfig: providerConfig,
	}, nil
}

func LoadTFExecVars(config *Config) *tfexec.VarOption {

	token := os.Getenv("HETZNER_TOKEN")
	return tfexec.Var(fmt.Sprintf("hcloud_token=%s", token))
}

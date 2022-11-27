package conf

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	conf, err := Load(filepath.Join("..", "testdata", "config.yaml"))
	assert.NoError(t, err)
	assert.NotNil(t, conf)
	fmt.Println(conf)

	assert.Equal(t, "config", conf.BaseDir)
	assert.Equal(t, "hetzner", conf.DC)
	assert.Equal(t, "ens10", conf.CloudProviderConfig.NetworkInterface)
	assert.Equal(t, "root", conf.CloudProviderConfig.User)
	assert.Equal(t, "hetzner", conf.CloudProviderConfig.Provider)

	assert.Equal(t, "loki", conf.ObservabilityConfig.LokiBucket)
	assert.Equal(t, "tempo", conf.ObservabilityConfig.TempoBucket)
	assert.Equal(t, false, conf.ObservabilityConfig.MultiInstance)

	assert.Equal(t, 2, conf.ClusterConfig.Clients)
	assert.Equal(t, 3, conf.ClusterConfig.Servers)
	assert.Equal(t, false, conf.ClusterConfig.SeparateConsulServers)
}

func TestLoadProviderConfig(t *testing.T) {
	conf, err := Load(filepath.Join("..", "testdata", "config.yaml"))
	assert.NoError(t, err)
	assert.NotNil(t, conf)

	provider, err := LoadTFVarsConfig(*conf)
	assert.NoError(t, err)
	assert.NotNil(t, provider)
	hetzner := provider.ProviderConfig.(HetznerSettings)

	expected := HetznerSettings{
		AllowedIPs:                []string{"85.4.84.201/32"},
		SSHKeys:                   []string{"wille.faler@gmail.com"},
		ServerInstanceType:        "cx21",
		ClientInstanceType:        "cx21",
		ObservabilityInstanceType: "cx21",
		Location:                  "nbg1",
		ResourceNames: HetznerResourceNames{
			BaseServerName: "nomad-srv",
			FirewallName:   "dev_firewall",
			NetworkName:    "dev_network",
		},
	}

	assert.Equal(t, expected, hetzner)
}

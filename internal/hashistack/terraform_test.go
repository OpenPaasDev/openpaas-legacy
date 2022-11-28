package hashistack

import (
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenPaaSDev/openpaas/internal/conf"
	"github.com/OpenPaaSDev/openpaas/internal/util"
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hclparse"
	"github.com/stretchr/testify/assert"
	"github.com/zclconf/go-cty/cty"
)

type tfConfig struct {
	Variable []Variable `hcl:"variable,block"`
}

type Variable struct {
	Type      *hcl.Attribute `hcl:"type"`
	Name      string         `hcl:"name,label"`
	Default   *cty.Value     `hcl:"default,optional"`
	Sensitive bool           `hcl:"sensitive,optional"`
}

func TestGenerateTerraform(t *testing.T) {
	config, err := conf.Load("../testdata/config.yaml")
	assert.NoError(t, err)

	folder := util.RandString(8)
	config.BaseDir = folder
	defer func() {
		e := os.RemoveAll(filepath.Clean(folder))
		assert.NoError(t, e)
	}()

	err = GenerateTerraform(config)
	assert.NoError(t, err)

	parser := hclparse.NewParser()
	f, parseDiags := parser.ParseHCLFile(filepath.Clean(filepath.Join(folder, "terraform", "vars.tf")))
	assert.False(t, parseDiags.HasErrors())

	_, parseDiags = parser.ParseHCLFile(filepath.Clean(filepath.Join(folder, "terraform", "main.tf")))
	assert.False(t, parseDiags.HasErrors())

	var conf tfConfig
	decodeDiags := gohcl.DecodeBody(f.Body, nil, &conf)
	assert.False(t, decodeDiags.HasErrors())

	vars := []struct {
		name       string
		tpe        string
		defaultVal cty.Value
	}{
		{name: "hcloud_token", tpe: "string", defaultVal: cty.NullVal(cty.String)},
		{name: "server_count", tpe: "number", defaultVal: cty.NumberVal(big.NewFloat(3))},
		{name: "client_count", tpe: "number", defaultVal: cty.NumberVal(big.NewFloat(2))},
		{name: "vault_count", tpe: "number", defaultVal: cty.NumberVal(big.NewFloat(2))},
		{name: "separate_consul_servers", tpe: "bool", defaultVal: cty.BoolVal(false)},
		{name: "multi_instance_observability", tpe: "bool", defaultVal: cty.BoolVal(false)},
		{name: "ssh_keys", tpe: "list", defaultVal: cty.TupleVal([]cty.Value{cty.StringVal("wille.faler@gmail.com")})},
		{name: "base_server_name", tpe: "string", defaultVal: cty.StringVal("nomad-srv")},
		{name: "firewall_name", tpe: "string", defaultVal: cty.StringVal("dev_firewall")},
		{name: "network_name", tpe: "string", defaultVal: cty.StringVal("dev_network")},
		{name: "allow_ips", tpe: "list", defaultVal: cty.TupleVal([]cty.Value{cty.StringVal("85.4.84.201/32")})},
		{name: "server_instance_type", tpe: "string", defaultVal: cty.StringVal("cx21")},
		{name: "location", tpe: "string", defaultVal: cty.StringVal("nbg1")},
	}

	expectedMap := make(map[string]string)
	for _, v := range conf.Variable {
		for _, expected := range vars {
			if expected.name == v.Name {
				expectedMap[expected.name] = expected.name
				assert.Equal(t, expected.tpe, v.Type.Expr.Variables()[0].RootName())
				if expected.defaultVal != cty.NullVal(cty.String) && !strings.Contains(expected.name, "_count") {
					assert.Equal(t, expected.defaultVal, *v.Default)
				}
				if strings.Contains(expected.name, "_count") {
					assert.Equal(t, expected.defaultVal.AsBigFloat().String(), v.Default.AsBigFloat().String())
				}
			}
		}
	}

	assert.Equal(t, len(expectedMap), len(vars))
}

package hashistack

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"os"
	"path/filepath"

	"github.com/OpenPaas/openpaas/internal/conf"
)

//go:embed templates/terraform/hetzner/main.tf
var hetznerMain string

//go:embed templates/terraform/hetzner/vars.tf
var hetznerVars string

func GenerateTerraform(config *conf.Config) error {
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

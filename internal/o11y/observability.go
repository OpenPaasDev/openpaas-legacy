package o11y

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/OpenPaas/openpaas/internal/ansible"
	"github.com/OpenPaas/openpaas/internal/conf"
	"github.com/OpenPaas/openpaas/internal/hashistack"
	"github.com/OpenPaas/openpaas/internal/runtime"
	"github.com/OpenPaas/openpaas/internal/secrets"
)

//go:embed templates/consul/intention.hcl
var consulIntention string

//go:embed templates/prometheus/prometheus.service
var prometheusService string

//go:embed templates/prometheus/install-prometheus.sh
var prometheusInstall string

//go:embed templates/prometheus/prometheus.yml
var prometheusYml string

//go:embed templates/loki/setup-loki-agent.sh
var lokiDockerAgent string

//go:embed templates/loki/loki.service
var lokiService string

//go:embed templates/loki/loki-config.yml
var lokiConfig string

//go:embed templates/loki/promtail.yml
var promtailConf string

//go:embed templates/loki/promtail.service
var promtailService string

//go:embed templates/ansible/observability.yml
var observabilityAnsible string

//go:embed templates/tempo/setup-tempo.sh
var tempoInstall string

//go:embed templates/tempo/tempo.service
var tempoService string

//go:embed templates/tempo/tempo.yml
var tempoConfig string

//go:embed templates/tempo/tempo-grpc.hcl
var tempoGrpcService string

//go:embed templates/tempo/tempo.hcl
var tempoConsulService string

//go:embed templates/loki/loki-http.hcl
var lokiHttpService string

//go:embed templates/consul/consul-ingress.hcl
var consulHttpService string

//go:embed templates/grafana/grafana.hcl
var grafanaHttpService string

//go:embed templates/prometheus/prometheus.hcl
var prometheusConsulService string

type consulServiceConf struct {
	template      string
	getPrivateIPs func(*ansible.Inventory) []string
	file          string
	name          string
}

func Init(config *conf.Config, inventory, configFile string, sec *secrets.Config) error {
	baseDir := config.BaseDir
	user := config.CloudProviderConfig.User
	inv, err := ansible.LoadInventory(inventory)
	if err != nil {
		return err
	}

	consul := hashistack.NewConsul(inv, sec, baseDir)

	err = mkObservabilityConfigs(consul, config, inv)
	if err != nil {
		return err
	}

	secretsFile := filepath.Join(baseDir, "secrets", "secrets.yaml")

	setup := filepath.Join(baseDir, "observability.yml")

	err = runtime.Exec(&runtime.EmptyEnv{}, fmt.Sprintf("ansible-playbook %s -i %s -u %s -e @%s -e @%s", setup, inventory, user, secretsFile, configFile), os.Stdout)
	if err != nil {
		return err
	}

	return nil
}

func mkObservabilityConfigs(consul hashistack.Consul, config *conf.Config, inv *ansible.Inventory) error {
	dirs := []string{
		"prometheus", "loki", "grafana", "intentions", "tempo", "consul",
	}
	baseDir := config.BaseDir
	for _, dir := range dirs {
		err := os.MkdirAll(filepath.Join(baseDir, dir), 0750)
		if err != nil {
			return err
		}
	}
	toWrite := map[string]string{
		filepath.Join(baseDir, "prometheus", "prometheus.service"):    prometheusService,
		filepath.Join(baseDir, "prometheus", "install-prometheus.sh"): prometheusInstall,
		filepath.Join(baseDir, "observability.yml"):                   observabilityAnsible,
		filepath.Join(baseDir, "loki", "setup-loki-agent.sh"):         lokiDockerAgent,
		filepath.Join(baseDir, "loki", "loki.service"):                lokiService,
		filepath.Join(baseDir, "loki", "promtail.service"):            promtailService,
		filepath.Join(baseDir, "loki", "loki-config.yml"):             lokiConfig,
		filepath.Join(baseDir, "tempo", "tempo.yml"):                  tempoConfig,
		filepath.Join(baseDir, "tempo", "tempo.service"):              tempoService,
		filepath.Join(baseDir, "tempo", "setup-tempo.sh"):             tempoInstall,
		filepath.Join(baseDir, "loki", "promtail.yml"):                promtailConf,
	}
	for k, v := range toWrite {
		err := os.WriteFile(k, []byte(v), 0600)
		if err != nil {
			return err
		}
	}

	clients := inv.All.Children.Clients.GetPrivateHosts()
	consulServers := inv.All.Children.ConsulServers.GetPrivateHosts()
	nomadServers := inv.All.Children.NomadServers.GetPrivateHosts()
	vaultServers := inv.All.Children.VaultServers.GetPrivateHosts()
	tempo := inv.All.Children.Tempo.GetPrivateHosts()
	prometheus := inv.All.Children.Prometheus.GetPrivateHosts()
	loki := inv.All.Children.Loki.GetPrivateHosts()
	grafana := inv.All.Children.Grafana.GetPrivateHosts()
	allObs := append(append(append(tempo, prometheus...), loki...), grafana...)

	obsKV := make(map[string]string)
	for _, server := range allObs {
		obsKV[server] = server
	}
	observabilityServers := []string{}
	for k := range obsKV {
		observabilityServers = append(observabilityServers, k)
	}

	tmpl, e := template.New("consul-policies").Parse(prometheusYml)
	if e != nil {
		return e
	}
	var buf bytes.Buffer

	err := tmpl.Execute(&buf, map[string]interface{}{
		"ConsulHosts": append(clients, consulServers...),
		"NomadHosts":  append(clients, nomadServers...),
		"AllHosts":    append(append(append(append(clients, nomadServers...), consulServers...), vaultServers...), observabilityServers...),
		"ConsulToken": "{{PROMETHEUS_CONSUL_TOKEN}}",
	})
	if err != nil {
		return err
	}

	output := buf.Bytes()

	err = os.WriteFile(filepath.Join(baseDir, "prometheus", "prometheus.yml"), []byte(output), 0600)
	if err != nil {
		return err
	}

	consulServices := []consulServiceConf{
		{
			template: tempoGrpcService,
			getPrivateIPs: func(inv *ansible.Inventory) []string {
				return inv.All.Children.Tempo.GetPrivateHosts()
			},
			file: filepath.Join(baseDir, "tempo", "tempo-grpc.hcl"),
			name: "tempo-grpc",
		},
		{
			template: tempoConsulService,
			getPrivateIPs: func(inv *ansible.Inventory) []string {
				return inv.All.Children.Tempo.GetPrivateHosts()
			},
			file: filepath.Join(baseDir, "tempo", "tempo.hcl"),
			name: "tempo",
		},
		{
			template: prometheusConsulService,
			getPrivateIPs: func(inv *ansible.Inventory) []string {
				return inv.All.Children.Prometheus.GetPrivateHosts()
			},
			file: filepath.Join(baseDir, "prometheus", "prometheus.hcl"),
			name: "prometheus",
		},
		{
			template: lokiHttpService,
			getPrivateIPs: func(inv *ansible.Inventory) []string {
				return inv.All.Children.Loki.GetPrivateHosts()
			},
			file: filepath.Join(baseDir, "loki", "loki.hcl"),
			name: "loki",
		},
		{
			template: grafanaHttpService,
			getPrivateIPs: func(inv *ansible.Inventory) []string {
				return inv.All.Children.Grafana.GetPrivateHosts()
			},
			file: filepath.Join(baseDir, "grafana", "grafana.hcl"),
			name: "grafana",
		},
		{
			template: consulHttpService,
			getPrivateIPs: func(inv *ansible.Inventory) []string {
				return inv.All.Children.ConsulServers.GetPrivateHosts()
			},
			file: filepath.Join(baseDir, "consul", "consul-ingress.hcl"),
			name: "consul-ingress",
		},
	}

	for _, service := range consulServices {
		servers := service.getPrivateIPs(inv)
		fmt.Println(servers)
		template := strings.ReplaceAll(service.template, "HOST", servers[0])
		template = strings.ReplaceAll(template, "ROOTDOMAIN", config.ClusterConfig.Ingress.ManagementDomain)
		err = os.WriteFile(filepath.Clean(service.file), []byte(template), 0600)
		if err != nil {
			return err
		}
		intention := strings.ReplaceAll(consulIntention, "SRVC", service.name)
		intentionFile := filepath.Join(baseDir, "intentions", fmt.Sprintf("%s.hcl", service.name))
		err = os.WriteFile(intentionFile, []byte(intention), 0600)
		if err != nil {
			return err
		}

		err = consul.RegisterService(service.file)
		if err != nil {
			return err
		}

		err = consul.RegisterIntention(intentionFile)
		if err != nil {
			return err
		}

	}

	return nil
}

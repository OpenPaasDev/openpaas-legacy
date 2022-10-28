package secrets

type Config struct {
	ConsulGossipKey        string       `yaml:"CONSUL_GOSSIP_KEY"`
	NomadGossipKey         string       `yaml:"NOMAD_GOSSIP_KEY"`
	NomadClientConsulToken string       `yaml:"NOMAD_CLIENT_CONSUL_TOKEN"`
	NomadServerConsulToken string       `yaml:"NOMAD_SERVER_CONSUL_TOKEN"`
	ConsulAgentToken       string       `yaml:"CONSUL_AGENT_TOKEN"`
	ConsulBootstrapToken   string       `yaml:"CONSUL_BOOTSTRAP_TOKEN"`
	PrometheusConsulToken  string       `yaml:"PROMETHEUS_CONSUL_TOKEN"`
	FabioConsulToken       string       `yaml:"FABIO_CONSUL_TOKEN"`
	VaultConsulToken       string       `yaml:"VAULT_CONSUL_TOKEN"`
	S3Endpoint             string       `yaml:"s3_endpoint"`
	S3AccessKey            string       `yaml:"s3_access_key"`
	S3SecretKey            string       `yaml:"s3_secret_key"`
	VaultConfig            VaultSecrets `yaml:"vault"`
}

type VaultSecrets struct {
	RootToken      string   `yaml:"root_token"`
	UnsealKeys     []string `yaml:"unseal_keys"`
	NomadRootToken string   `yaml:"nomad_root_token"`
}

package hashistack

import (
	"fmt"
	"os"

	"github.com/OpenPaaSDev/openpaas/internal/runtime"
)

type NomadClient interface {
	RunJob(jobFile string) error
}

type nomadCli struct {
	baseDir, nomadAddr, caCert, clientCert, clientKey string
}

func NewNomadClient(baseDir, nomadAddr, caCert, clientCert, clientKey string) NomadClient {
	return &nomadCli{
		baseDir:    baseDir,
		nomadAddr:  nomadAddr,
		caCert:     caCert,
		clientCert: clientCert,
		clientKey:  clientKey,
	}
}

func (nomad *nomadCli) Get() map[string]string {
	return map[string]string{
		"NOMAD_ADDR":        nomad.nomadAddr,
		"NOMAD_CACERT":      nomad.caCert,
		"NOMAD_CLIENT_CERT": nomad.clientCert,
		"NOMAD_CLIENT_KEY":  nomad.clientKey,
	}
}

func (nomad *nomadCli) WorkingDir() string {
	return nomad.baseDir
}

func (nomad *nomadCli) RunJob(jobFile string) error {
	cmd := fmt.Sprintf("nomad job run %s", jobFile)
	return runtime.Exec(nomad, cmd, os.Stdout)
}

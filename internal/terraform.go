package internal

import (
	"context"
	"io"

	"github.com/hashicorp/go-version"
	install "github.com/hashicorp/hc-install"
	"github.com/hashicorp/hc-install/fs"
	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/hc-install/releases"
	"github.com/hashicorp/hc-install/src"
	"github.com/hashicorp/terraform-exec/tfexec"
)

func InitTf(ctx context.Context, workingDir string, stdOut, stdErr io.Writer) (*tfexec.Terraform, error) {

	i := install.NewInstaller()

	version := version.Must(version.NewVersion("1.2.6"))

	execPath, err := i.Ensure(ctx, []src.Source{
		&fs.ExactVersion{
			Product: product.Terraform,
			Version: version,
		},
		&releases.ExactVersion{
			Product: product.Terraform,
			Version: version,
		},
	})
	if err != nil {
		return nil, err
	}

	tf, err := tfexec.NewTerraform(workingDir, execPath)
	if err != nil {
		return nil, err
	}

	err = tf.Init(ctx, tfexec.Upgrade(true))
	if err != nil {
		return nil, err
	}

	tf.SetStdout(stdOut)
	tf.SetStderr(stdErr)
	return tf, nil
}

package ansible

import (
	"os/user"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAnsible(t *testing.T) {
	currentUser, err := user.Current()
	assert.NoError(t, err)

	ansibleClient := NewClient(filepath.Join("testdata", "inventory"), filepath.Join("testdata", "secrets"), currentUser.Username, filepath.Join("testdata", "secrets"))
	err = ansibleClient.Run(filepath.Join("testdata", "ansible.yml"))
	assert.NoError(t, err)
}

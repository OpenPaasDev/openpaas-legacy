package ansible

import (
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenPaaSDev/openpaas/internal/conf"
	"github.com/OpenPaaSDev/openpaas/internal/util"
	"github.com/stretchr/testify/assert"
)

func TestGenerateInventory(t *testing.T) {
	config, err := conf.Load(filepath.Join("testdata", "config.yaml"))
	assert.NoError(t, err)

	folder := util.RandString(8)
	config.BaseDir = folder
	err = os.MkdirAll(folder, 0700)
	assert.NoError(t, err)
	defer func() {
		e := os.RemoveAll(filepath.Join(folder))
		assert.NoError(t, e)
	}()
	src := filepath.Join("testdata", "inventory-output.json")
	dest := filepath.Join(folder, "inventory-output.json")

	bytesRead, err := os.ReadFile(filepath.Clean(src))

	if err != nil {
		log.Fatal(err)
	}

	err = os.WriteFile(dest, bytesRead, 0600)

	if err != nil {
		log.Fatal(err)
	}

	inventory, err := GenerateInventory(config)
	assert.NoError(t, err)

	assert.FileExists(t, filepath.Join(folder, "inventory"))

	assert.NotEmpty(t, inventory.GetAllPrivateHosts())

	_, err = LoadInventory(filepath.Join(folder, "inventory"))
	assert.NoError(t, err)

	assert.NotEmpty(t, inventory.All.Children.Clients.GetHosts())

}

package ansible

// func TestGenerateInventory(t *testing.T) {
// 	config, err := LoadConfig("testdata/config.yaml")
// 	assert.NoError(t, err)

// 	folder := RandString(8)
// 	config.BaseDir = folder
// 	err = os.MkdirAll(folder, 0700)
// 	assert.NoError(t, err)
// 	defer func() {
// 		e := os.RemoveAll(filepath.Join(folder))
// 		assert.NoError(t, e)
// 	}()

// 	src := filepath.Join("testdata", "inventory.json")
// 	dest := filepath.Join(folder, "inventory-output.json")

// 	bytesRead, err := os.ReadFile(filepath.Clean(src))
// 	assert.NoError(t, err)
// 	fmt.Println(string(bytesRead))

// 	err = os.WriteFile(filepath.Clean(dest), bytesRead, 0600)
// 	assert.NoError(t, err)

// 	err = GenerateInventory(config)
// 	assert.NoError(t, err)
// 	bytesRead, err = os.ReadFile(filepath.Clean(filepath.Join(folder, "inventory")))
// 	assert.NoError(t, err)
// 	assert.Equal(t, inventoryResultTest, string(bytesRead))
// }

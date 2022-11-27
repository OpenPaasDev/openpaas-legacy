package vault

import (
	"path/filepath"
	"testing"

	"github.com/OpenPaas/openpaas/internal/secrets"
	"github.com/stretchr/testify/assert"
)

func TestParseInit(t *testing.T) {
	secret, err := parseVaultInit(filepath.Join("testdata", "operator-init.txt"), &secrets.Config{})
	assert.NoError(t, err)

	assert.NotNil(t, secret)

	assert.Equal(t, "hvs.tswBMw2mcJv9cLkQ7cfsxDZg", secret.VaultConfig.RootToken)

	assert.Equal(t, []string{
		"4IwfRgraGxXvnBKTkW6hMt5S+pPWFnKG9WXYQJBCbDV4",
		"rLlg7MZnchlpz3NxRWMjj0joiH7qs++hLxHGefyQ4Rm7",
		"+ZzoQ4KYENet1D+qRZMmXuCCUmdKOjbCybjWeSM7PQtg",
		"qfaVNHwa1L1IllpYS8OY9xqQyhYbNGlHZ6pyOL2fBCtW",
		"ew4QyHy+YNGxWWjNIiN5RzfZ8+k92goN5D9+MmKX8y9k",
	}, secret.VaultConfig.UnsealKeys)
}

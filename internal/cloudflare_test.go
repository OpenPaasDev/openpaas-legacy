package internal

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetCloudflareIPs(t *testing.T) {

	ips, err := GetCloudflareIPs(context.Background())
	assert.NoError(t, err)
	assert.NotEmpty(t, ips.IPV4)
	assert.NotEmpty(t, ips.IPV6)
	assert.NotEqual(t, ips.IPV4, ips.IPV6)

}

package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestComposeWiresOnlineConfPrivately(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile(filepath.Join("..", "..", "docker-compose.yaml"))
	require.NoError(t, err)
	var document struct {
		Services map[string]struct {
			DependsOn   map[string]any `yaml:"depends_on"`
			Environment map[string]any `yaml:"environment"`
			Ports       []string       `yaml:"ports"`
			Healthcheck map[string]any `yaml:"healthcheck"`
		} `yaml:"services"`
	}
	require.NoError(t, yaml.Unmarshal(raw, &document))

	backend, ok := document.Services["backend"]
	require.True(t, ok)
	require.Contains(t, backend.DependsOn, "onlineconf")
	require.Equal(t, "http://onlineconf:8081", backend.Environment["ONLINECONF_URL"])

	service, ok := document.Services["onlineconf"]
	require.True(t, ok)
	require.NotEmpty(t, service.Healthcheck)
	require.Len(t, service.Ports, 1)
	require.True(t, strings.HasPrefix(service.Ports[0], "127.0.0.1:"), "OnlineConf must not bind on every host interface")
}

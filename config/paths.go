package config

import (
	"fmt"
	"os"
	"path/filepath"
)

var (
	HomeDir    = os.Getenv("HOME")
	KubeDir    = filepath.Join(HomeDir, ".kube")
	ConfigFile = filepath.Join(KubeDir, "config.json")

	// Session and cache directories
	SessionDir = filepath.Join(KubeDir, "sessions")
	CacheDir   = filepath.Join(KubeDir, "cache")

	// Active kubeconfig file
	KubeConfigFile = filepath.Join(KubeDir, "config")
)

// GetSessionConfig returns the path for a session-specific kubeconfig
func GetSessionConfig(name string) string {
	return filepath.Join(SessionDir, fmt.Sprintf("%s-%d.config", name, os.Getpid()))
}

// SetKubeconfig sets the KUBECONFIG environment variable
func SetKubeconfig(path string) error {
	return os.Setenv("KUBECONFIG", path)
}

// ResetKubeconfig unsets the KUBECONFIG environment variable
func ResetKubeconfig() error {
	return os.Unsetenv("KUBECONFIG")
}

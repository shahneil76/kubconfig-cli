package config

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v2"
)

type TokenManager struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

type KubeToken struct {
	APIVersion string            `yaml:"apiVersion"`
	Kind       string            `yaml:"kind"`
	Metadata   map[string]string `yaml:"metadata"`
	Spec       TokenSpec         `yaml:"spec"`
}

type TokenSpec struct {
	ExpirationSeconds int64 `yaml:"expirationSeconds"`
}

// Generate a temporary token with expiry
func GenerateToken(duration time.Duration) *TokenManager {
	// Convert duration to seconds
	seconds := int64(duration.Seconds())

	// Create a token request
	tokenReq := &KubeToken{
		APIVersion: "authentication.k8s.io/v1",
		Kind:       "TokenRequest",
		Metadata: map[string]string{
			"name": "temp-token-" + randString(8),
		},
		Spec: TokenSpec{
			ExpirationSeconds: seconds,
		},
	}

	// Convert to YAML
	tokenData, _ := yaml.Marshal(tokenReq)

	return &TokenManager{
		Token:     base64.StdEncoding.EncodeToString(tokenData),
		ExpiresAt: time.Now().Add(duration),
	}
}

// ModifyKubeconfig replaces the auth token in the kubeconfig
func ModifyKubeconfig(kubeconfigData []byte, tokenManager *TokenManager) ([]byte, error) {
	var kubeconfig yaml.MapSlice
	if err := yaml.Unmarshal(kubeconfigData, &kubeconfig); err != nil {
		return nil, err
	}

	// Find users section
	for _, item := range kubeconfig {
		if item.Key == "users" {
			if users, ok := item.Value.([]interface{}); ok {
				for _, user := range users {
					if userMap, ok := user.(map[interface{}]interface{}); ok {
						if auth, ok := userMap["user"].(map[interface{}]interface{}); ok {
							// Create a temporary token with expiry
							tempToken := fmt.Sprintf("temp-%s-%d",
								randString(8),
								tokenManager.ExpiresAt.Unix(),
							)

							// Store original token and expiry in cache
							if err := cacheCredentials(tempToken, auth["token"], tokenManager.ExpiresAt); err != nil {
								return nil, err
							}

							// Replace with temporary token
							auth["token"] = tempToken
							delete(auth, "client-certificate-data")
							delete(auth, "client-key-data")
						}
					}
				}
			}
		}
	}

	return yaml.Marshal(kubeconfig)
}

// Store credentials in cache
func cacheCredentials(tempToken interface{}, origToken interface{}, expiry time.Time) error {
	creds := struct {
		OriginalToken string    `json:"original_token"`
		ExpiresAt     time.Time `json:"expires_at"`
	}{
		OriginalToken: fmt.Sprintf("%v", origToken),
		ExpiresAt:     expiry,
	}

	data, err := json.Marshal(creds)
	if err != nil {
		return err
	}

	cacheFile := filepath.Join(CacheDir, fmt.Sprintf("%v.creds", tempToken))
	return os.WriteFile(cacheFile, data, 0600)
}

// GetOriginalToken verifies and returns the original token
func GetOriginalToken(tempToken string) (string, error) {
	cacheFile := filepath.Join(CacheDir, fmt.Sprintf("%s.creds", tempToken))
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return "", err
	}

	var creds struct {
		OriginalToken string    `json:"original_token"`
		ExpiresAt     time.Time `json:"expires_at"`
	}

	if err := json.Unmarshal(data, &creds); err != nil {
		return "", err
	}

	if time.Now().After(creds.ExpiresAt) {
		os.Remove(cacheFile)
		return "", fmt.Errorf("token expired")
	}

	return creds.OriginalToken, nil
}

func randString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// VerifyTokenExpiry checks if the token has expired
func VerifyTokenExpiry(kubeconfigPath string) (bool, time.Time, error) {
	data, err := os.ReadFile(kubeconfigPath)
	if err != nil {
		return false, time.Time{}, err
	}

	var kubeconfig yaml.MapSlice
	if err := yaml.Unmarshal(data, &kubeconfig); err != nil {
		return false, time.Time{}, err
	}

	for _, item := range kubeconfig {
		if item.Key == "users" {
			if users, ok := item.Value.([]interface{}); ok {
				for _, user := range users {
					if userMap, ok := user.(map[interface{}]interface{}); ok {
						if auth, ok := userMap["user"].(map[interface{}]interface{}); ok {
							// Check for exec configuration
							if execConfig, ok := auth["exec"].(map[interface{}]interface{}); ok {
								if args, ok := execConfig["args"].([]interface{}); ok {
									for i, arg := range args {
										if arg == "--expiry" && i+1 < len(args) {
											if expiryStr, ok := args[i+1].(string); ok {
												expiry, err := time.Parse(time.RFC3339, expiryStr)
												if err != nil {
													return false, time.Time{}, err
												}
												return time.Now().After(expiry), expiry, nil
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
	return false, time.Time{}, fmt.Errorf("no token expiry found")
}

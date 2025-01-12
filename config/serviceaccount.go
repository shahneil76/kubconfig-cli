package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"text/template"
	"time"

	"encoding/base64"

	"gopkg.in/yaml.v3"
)

type ServiceAccountConfig struct {
	Name        string
	Namespace   string
	ServerURL   string
	ClusterName string
	User        string
	ExpiresAt   time.Time
	CreatedAt   string
}

const serviceAccountTemplate = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: {{ .Name }}
  namespace: {{ .Namespace }}
  labels:
    kubconfig.io/managed-by: "kubconfig-cli"
    kubconfig.io/user: "{{ .User }}"
  annotations:
    kubconfig.io/created-by: "{{ .User }}"
    kubconfig.io/created-at: "{{ .CreatedAt }}"
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: {{ .Name }}-admin
  labels:
    kubconfig.io/managed-by: "kubconfig-cli"
    kubconfig.io/user: "{{ .User }}"
  annotations:
    kubconfig.io/created-by: "{{ .User }}"
    kubconfig.io/created-at: "{{ .CreatedAt }}"
subjects:
- kind: ServiceAccount
  name: {{ .Name }}
  namespace: {{ .Namespace }}
roleRef:
  kind: ClusterRole
  name: cluster-admin
  apiGroup: rbac.authorization.k8s.io
`

// Add this to track active sessions
var (
	activeSessionsMutex sync.RWMutex
	activeSessions      = make(map[string]*ServiceAccountConfig)
)

func verifyClusterAccess() error {
	cmd := exec.Command("kubectl", "auth", "can-i", "create", "serviceaccount")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("no cluster access: %v\nkubectl output: %s", err, stderr.String())
	}
	return nil
}

func CreateTemporaryAccess(duration time.Duration) (*ServiceAccountConfig, error) {
	fmt.Println("Creating temporary access...")

	// First verify cluster access
	if err := verifyClusterAccess(); err != nil {
		return nil, fmt.Errorf("cluster access check failed: %v", err)
	}

	// Get current user
	user, err := getCurrentUser()
	if err != nil {
		return nil, err
	}

	// Get cluster info
	clusterName, serverURL, err := getClusterInfo()
	if err != nil {
		return nil, err
	}

	config := &ServiceAccountConfig{
		Name:        fmt.Sprintf("%s-user", user),
		Namespace:   "kube-system",
		ServerURL:   serverURL,
		ClusterName: clusterName,
		User:        user,
		ExpiresAt:   time.Now().Add(duration),
		CreatedAt:   time.Now().Format(time.RFC3339),
	}

	// Check if SA already exists
	if !serviceAccountExists(config) {
		// Create service account and related resources
		if err := createResources(config); err != nil {
			return nil, err
		}

		// Wait for SA to be ready
		if err := waitForServiceAccount(config); err != nil {
			return nil, fmt.Errorf("service account not ready: %v", err)
		}
	}

	fmt.Println("Temporary access created successfully.")
	return config, nil
}

func waitForServiceAccount(config *ServiceAccountConfig) error {
	maxAttempts := 10
	for i := 0; i < maxAttempts; i++ {
		cmd := exec.Command("kubectl", "get", "serviceaccount",
			config.Name,
			"-n", config.Namespace,
			"-o", "jsonpath={.metadata.name}")

		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		output, err := cmd.Output()
		if err == nil && len(output) > 0 {
			return nil
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("timeout waiting for service account to be ready")
}

func serviceAccountExists(config *ServiceAccountConfig) bool {
	cmd := exec.Command("kubectl", "get", "serviceaccount",
		config.Name,
		"-n", config.Namespace,
		"--ignore-not-found",
		"-o", "jsonpath={.metadata.name}")
	output, _ := cmd.Output()
	return len(output) > 0
}

func createResources(config *ServiceAccountConfig) error {
	tmpl, err := template.New("sa").Parse(serviceAccountTemplate)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, config); err != nil {
		return err
	}

	// Apply the configuration
	cmd := exec.Command("kubectl", "apply", "-f", "-")
	cmd.Stdin = &buf
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// Debug: Print the resources being created
	fmt.Printf("Creating resources:\n%s\n", buf.String())

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error creating resources: %v\nkubectl output: %s", err, stderr.String())
	}

	return nil
}

func CleanupTemporaryAccess(config *ServiceAccountConfig) error {
	// Check if there are any other active sessions for this user
	activeSessionsMutex.RLock()
	hasActiveSessions := false
	for _, session := range activeSessions {
		if session.User == config.User && !time.Now().After(session.ExpiresAt) {
			hasActiveSessions = true
			break
		}
	}
	activeSessionsMutex.RUnlock()

	// Don't clean up if there are other active sessions
	if hasActiveSessions {
		return nil
	}

	var errs []string

	// Delete in reverse order
	cmd := exec.Command("kubectl", "delete", "clusterrolebinding",
		fmt.Sprintf("%s-admin", config.Name),
		"--ignore-not-found=true")
	if err := cmd.Run(); err != nil {
		errs = append(errs, fmt.Sprintf("failed to delete clusterrolebinding: %v", err))
	}

	cmd = exec.Command("kubectl", "delete", "serviceaccount",
		"-n", config.Namespace,
		config.Name,
		"--ignore-not-found=true")
	if err := cmd.Run(); err != nil {
		errs = append(errs, fmt.Sprintf("failed to delete serviceaccount: %v", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("cleanup errors: %s", strings.Join(errs, "; "))
	}
	return nil
}

func getCurrentUser() (string, error) {
	cmd := exec.Command("whoami")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(bytes.TrimSpace(out)), nil
}

func getClusterInfo() (string, string, error) {
	// First check if we can access the cluster
	cmd := exec.Command("kubectl", "cluster-info")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", "", fmt.Errorf("cannot access cluster: %v\nkubectl output: %s", err, stderr.String())
	}

	cmd = exec.Command("kubectl", "config", "view", "--minify", "-o", "jsonpath={.clusters[0].name}")
	clusterName, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("error getting cluster name: %v", err)
	}

	cmd = exec.Command("kubectl", "config", "view", "--minify", "-o", "jsonpath={.clusters[0].cluster.server}")
	serverURL, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("error getting server URL: %v", err)
	}

	if len(clusterName) == 0 || len(serverURL) == 0 {
		return "", "", fmt.Errorf("invalid cluster info: name=%q, url=%q", string(clusterName), string(serverURL))
	}

	return string(bytes.TrimSpace(clusterName)), string(bytes.TrimSpace(serverURL)), nil
}

func GetTokenAndCert(config *ServiceAccountConfig) (string, string, error) {
	// First get the CA cert from the cluster
	cmd := exec.Command("kubectl", "config", "view", "--raw", "-o",
		"jsonpath={.clusters[0].cluster.certificate-authority-data}")
	caBytes, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("error getting CA cert: %v", err)
	}

	// Calculate duration and ensure it's in seconds
	duration := time.Until(config.ExpiresAt).Round(time.Second)
	if duration < time.Second {
		return "", "", fmt.Errorf("duration must be at least 1 second")
	}

	// Create token with TTL
	tokenCmd := exec.Command("kubectl", "create", "token",
		config.Name,
		"--namespace", config.Namespace,
		"--duration", fmt.Sprintf("%ds", int(duration.Seconds()))) // Format duration in seconds

	var stderr bytes.Buffer
	tokenCmd.Stderr = &stderr
	tokenBytes, err := tokenCmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("error creating token: %v\nOutput: %s", err, stderr.String())
	}

	// Verify token expiry
	token := string(bytes.TrimSpace(tokenBytes))
	if err := verifyTokenExpiry(token, config.ExpiresAt); err != nil {
		return "", "", fmt.Errorf("token verification failed: %v", err)
	}

	fmt.Printf("Created token for service account: %s (expires in %s)\n",
		config.Name, duration.Round(time.Second))

	return token, string(caBytes), nil
}

// Add function to verify token expiry
func verifyTokenExpiry(token string, expectedExpiry time.Time) error {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return fmt.Errorf("invalid token format")
	}

	// Decode the payload (second part)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return fmt.Errorf("error decoding token: %v", err)
	}

	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return fmt.Errorf("error parsing token claims: %v", err)
	}

	tokenExpiry := time.Unix(claims.Exp, 0)
	fmt.Printf("Token will expire at: %s\n", tokenExpiry.Format(time.RFC3339))

	// Allow small time difference (1 minute) due to processing time
	if diff := tokenExpiry.Sub(expectedExpiry).Abs(); diff > time.Minute {
		return fmt.Errorf("token expiry mismatch: expected %s, got %s",
			expectedExpiry.Format(time.RFC3339),
			tokenExpiry.Format(time.RFC3339))
	}

	return nil
}

func GetServiceAccountFromConfig(configPath string) (*ServiceAccountConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var kubeconfig map[string]interface{}
	if err := yaml.Unmarshal(data, &kubeconfig); err != nil {
		return nil, err
	}

	// Extract service account name from context
	contexts, ok := kubeconfig["contexts"].([]interface{})
	if !ok || len(contexts) == 0 {
		return nil, fmt.Errorf("no contexts found")
	}

	context := contexts[0].(map[interface{}]interface{})
	contextName := context["name"].(string)
	parts := strings.Split(contextName, "@")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid context name format")
	}

	return &ServiceAccountConfig{
		Name:      parts[0],
		Namespace: "kube-system",
	}, nil
}

func GenerateKubeconfig(config *ServiceAccountConfig, token, caCert string) string {
	return fmt.Sprintf(`apiVersion: v1
kind: Config
clusters:
- name: %s
  cluster:
    certificate-authority-data: %s
    server: %s
contexts:
- name: %s@%s
  context:
    cluster: %s
    namespace: default
    user: %s
current-context: %s@%s
users:
- name: %s
  user:
    token: %s
`, config.ClusterName, caCert, config.ServerURL,
		config.Name, config.ClusterName,
		config.ClusterName, config.Name,
		config.Name, config.ClusterName,
		config.Name, token)
}

// Add a function to wait for secret creation
func WaitForSecret(config *ServiceAccountConfig) error {
	maxAttempts := 10
	for i := 0; i < maxAttempts; i++ {
		cmd := exec.Command("kubectl", "get", "secret",
			"-n", config.Namespace,
			"-l", fmt.Sprintf("kubernetes.io/service-account.name=%s", config.Name))
		if err := cmd.Run(); err == nil {
			return nil
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("timeout waiting for service account secret")
}

func StartCleanupRoutine() {
	go func() {
		for {
			time.Sleep(time.Minute)
			cleanupExpiredSessions()
		}
	}()
}

func cleanupExpiredSessions() {
	activeSessionsMutex.Lock()
	defer activeSessionsMutex.Unlock()

	now := time.Now()
	for id, session := range activeSessions {
		if now.After(session.ExpiresAt) {
			if err := CleanupTemporaryAccess(session); err != nil {
				fmt.Printf("Error cleaning up expired session %s: %v\n", id, err)
			}
			delete(activeSessions, id)
		}
	}
}

func RegisterSession(config *ServiceAccountConfig) {
	activeSessionsMutex.Lock()
	defer activeSessionsMutex.Unlock()
	activeSessions[config.Name] = config
}

func GetTemporaryToken(config *ServiceAccountConfig) (string, error) {
	// Wait for SA to be ready (double-check)
	if err := waitForServiceAccount(config); err != nil {
		return "", fmt.Errorf("service account not ready: %v", err)
	}

	// Calculate duration in seconds
	durationSeconds := int(time.Until(config.ExpiresAt).Seconds())

	// Enforce minimum duration of 10 minutes (600 seconds)
	minSeconds := 600
	if durationSeconds < minSeconds {
		fmt.Printf("Warning: Adjusting duration to minimum allowed (10 minutes)\n")
		durationSeconds = minSeconds
		config.ExpiresAt = time.Now().Add(10 * time.Minute)
	}

	// Show loading animation
	done := make(chan bool)
	go showLoadingAnimation("Creating token", done)

	// Create token with TTL
	tokenCmd := exec.Command("kubectl", "create", "token",
		config.Name,
		"--namespace", config.Namespace,
		"--duration", fmt.Sprintf("%ds", durationSeconds))

	var stderr bytes.Buffer
	tokenCmd.Stderr = &stderr
	tokenBytes, err := tokenCmd.Output()

	// Stop loading animation
	done <- true
	<-done

	if err != nil {
		return "", fmt.Errorf("error creating token: %v\nOutput: %s", err, stderr.String())
	}

	token := string(bytes.TrimSpace(tokenBytes))
	if err := verifyTokenExpiry(token, config.ExpiresAt); err != nil {
		return "", fmt.Errorf("token verification failed: %v", err)
	}

	// Format expiry time nicely
	duration := time.Duration(durationSeconds) * time.Second
	var expiryMsg string
	switch {
	case duration.Hours() >= 1:
		hours := int(duration.Hours())
		minutes := int(duration.Minutes()) % 60
		if minutes > 0 {
			expiryMsg = fmt.Sprintf("%d hour%s %d minute%s",
				hours, pluralize(hours),
				minutes, pluralize(minutes))
		} else {
			expiryMsg = fmt.Sprintf("%d hour%s", hours, pluralize(hours))
		}
	case duration.Minutes() >= 1:
		minutes := int(duration.Minutes())
		expiryMsg = fmt.Sprintf("%d minute%s", minutes, pluralize(minutes))
	default:
		expiryMsg = fmt.Sprintf("%d second%s", durationSeconds, pluralize(durationSeconds))
	}

	fmt.Printf("\n✓ Token created successfully (expires in %s)\n", expiryMsg)

	return token, nil
}

// Helper function to add plural 's'
func pluralize(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func showLoadingAnimation(message string, done chan bool) {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	i := 0
	for {
		select {
		case <-done:
			fmt.Print("\r\033[K") // Clear the line
			done <- true
			return
		default:
			fmt.Printf("\r%s %s", frames[i], message)
			i = (i + 1) % len(frames)
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func ModifyKubeconfigWithToken(originalConfig []byte, token string) ([]byte, error) {
	var kubeconfig map[string]interface{}
	if err := yaml.Unmarshal(originalConfig, &kubeconfig); err != nil {
		return nil, fmt.Errorf("error parsing kubeconfig: %v", err)
	}

	// Modify the users section to use the new token
	users, ok := kubeconfig["users"].([]interface{})
	if !ok || len(users) == 0 {
		return nil, fmt.Errorf("no users found in kubeconfig")
	}

	user := users[0].(map[string]interface{})
	userData := user["user"].(map[string]interface{})

	// Replace only the token, keeping other auth methods if present
	userData["token"] = token

	return yaml.Marshal(kubeconfig)
}

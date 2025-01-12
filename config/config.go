package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
)

type Config struct {
	S3Bucket       string `json:"s3_bucket"`
	Region         string `json:"region"`
	AWSAccessKey   string `json:"aws_access_key"`
	AWSSecretKey   string `json:"aws_secret_key"`
	S3Endpoint     string `json:"s3_endpoint"`
	ForcePathStyle bool   `json:"force_path_style"`
}

func SaveConfig(cfg Config) error {
	dir := filepath.Dir(ConfigFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	file, err := os.Create(ConfigFile)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(cfg)
}

func LoadConfig() (Config, error) {
	file, err := os.Open(ConfigFile)
	if err != nil {
		return Config{}, errors.New("CLI not configured. Run 'kubconfig init' first")
	}
	defer file.Close()

	var cfg Config
	if err := json.NewDecoder(file).Decode(&cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// ValidateKubeconfigName checks if the kubeconfig name is valid
func ValidateKubeconfigName(name string) error {
	if name == "" {
		return errors.New("kubeconfig name cannot be empty")
	}
	if !strings.HasSuffix(name, ".cfg") {
		return errors.New("kubeconfig name must end with .cfg")
	}
	return nil
}

// IsCached checks if a kubeconfig is already cached
func IsCached(configName string) bool {
	path := filepath.Join(CacheDir, configName)
	_, err := os.Stat(path)
	return err == nil
}

// FormatEndpointURL formats and validates the S3 endpoint URL
func FormatEndpointURL(endpoint string) string {
	if endpoint == "" {
		return ""
	}

	// Clean the URL
	endpoint = strings.TrimSpace(endpoint)
	endpoint = strings.TrimSuffix(endpoint, "/")

	// Add http:// if no protocol specified
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		endpoint = "http://" + endpoint
	}

	return endpoint
}

// CreateS3Session creates an AWS session with the given configuration
func CreateS3Session(cfg Config) (*session.Session, error) {
	awsConfig := &aws.Config{
		Region:           aws.String(cfg.Region),
		Credentials:      credentials.NewStaticCredentials(cfg.AWSAccessKey, cfg.AWSSecretKey, ""),
		S3ForcePathStyle: aws.Bool(cfg.ForcePathStyle),
	}

	if cfg.S3Endpoint != "" {
		endpoint := strings.TrimSuffix(cfg.S3Endpoint, "/")
		if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
			endpoint = "http://" + endpoint
		}
		awsConfig.Endpoint = aws.String(endpoint)
	}

	return session.NewSession(awsConfig)
}

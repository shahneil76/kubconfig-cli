# kubconfig-cli

A secure and efficient Kubernetes configuration manager with temporary access controls and centralized config management.

[![Go Report Card](https://goreportcard.com/badge/github.com/shahneil76/kubconfig-cli)](https://goreportcard.com/report/github.com/shahneil76/kubconfig-cli)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

## Overview

kubconfig-cli solves three critical challenges in Kubernetes configuration management:

1. **Secure Access Management**: Automatically creates temporary service accounts with TTL-based tokens, ensuring access is automatically revoked.
2. **Centralized Configuration**: Stores kubeconfig files in a central S3 location, making them easily accessible across teams.
3. **Session Management**: Provides temporary access sessions with automatic cleanup, preventing stale credentials.

## Key Features

- 🔐 **Temporary Access Control**
  - Creates time-limited service accounts
  - Auto-expiring tokens (minimum 10 minutes)
  - Automatic cleanup of expired resources

- 🎯 **Centralized Management**
  - S3-based kubeconfig storage
  - Compatible with any S3-compatible storage
  - Team-wide configuration sharing

- 🔄 **Session Management**
  - Duration-based access sessions
  - Automatic token refresh
  - Clean deactivation process

- 📊 **Cluster Analysis**
  - Resource usage monitoring
  - Health checks
  - Detailed cluster metrics

## Why kubconfig-cli?

kubconfig-cli was born from real-world challenges in managing Kubernetes access across teams and organizations. It addresses several key needs:

### Secure Temporary Access
Traditional kubeconfig files often contain long-lived credentials that can pose security risks if leaked or forgotten. kubconfig-cli transforms this by creating temporary, auto-expiring access credentials. When you activate a kubeconfig, it creates a time-limited service account with TTL-based tokens, ensuring access is automatically revoked after the session ends.

### Centralized Management
Organizations often struggle with kubeconfig sprawl - configuration files scattered across different machines, teams, and environments. kubconfig-cli solves this by storing master kubeconfig files in a central S3 location. Teams can access the same configurations while maintaining security through temporary credentials.

### Perfect For:

1. **Enterprise Security Teams**
   When security is paramount, kubconfig-cli helps enforce access policies by ensuring:
   - No permanent credentials are distributed
   - Access is automatically revoked
   - All access is auditable through service account tracking
   - Teams can't accidentally leak long-lived credentials

2. **DevOps Teams**
   For teams managing multiple clusters or environments:
   - Access any cluster without managing multiple kubeconfig files
   - Share access securely with team members
   - No need to manually revoke access or clean up credentials
   - Maintain security best practices without overhead

3. **Managed Service Providers**
   When providing Kubernetes services to customers:
   - Grant temporary access to customers securely
   - Maintain central control over all cluster access
   - Ensure access is automatically revoked
   - Keep master configurations secure while sharing access

The tool shines in environments where security and convenience need to coexist - where teams need quick access to clusters but security teams need to ensure that access is controlled, temporary, and auditable.

## Installation
I'll complete the README.md file from where it left off:

```markdown:README.md
## Installation

```bash
# Using go install
go install github.com/shahneil76/kubconfig-cli@latest

# Or download binary
curl -LO https://github.com/shahneil76/kubconfig-cli/releases/latest/download/kubconfig-$(uname -s)-$(uname -m)
chmod +x kubconfig-$(uname -s)-$(uname -m)
sudo mv kubconfig-$(uname -s)-$(uname -m) /usr/local/bin/kubconfig
```

## Quick Start

1. **Initialize with S3**:
```bash
kubconfig init
# Enter your S3 bucket details and credentials when prompted
```

2. **List Available Configs**:
```bash
kubconfig list
# Shows all kubeconfig files stored in S3
```

3. **Activate with Session**:
```bash
kubconfig activate dev-cluster.cfg --session 1h
# Creates temporary access for 1 hour
```

4. **Verify Access**:
```bash
kubconfig verify
# Checks cluster connectivity and permissions
```

5. **Monitor Status**:
```bash
kubconfig status
# Shows remaining session time
```

6. **Deactivate When Done**:
```bash
kubconfig deactivate
# Cleans up temporary resources
```

## Command Reference

### Basic Commands

- `init` - Configure S3 storage settings
- `list` - Show available kubeconfig files
- `activate` - Activate a kubeconfig with temporary access
- `deactivate` - Remove temporary access
- `status` - Check current session status
- `verify` - Verify cluster access

### Advanced Commands

- `analyze` - Show detailed cluster analysis
- `cleanup` - Clean up expired sessions
- `shell` - Configure shell integration

## Security Best Practices

1. **Session Duration**
   - Use minimum required duration
   - Default maximum: 24 hours
   - Recommended: 2-4 hours for regular use

2. **Access Management**
   - Regular cleanup of expired sessions
   - Verify access using `kubconfig verify`
   - Monitor active sessions with `kubconfig status`

3. **S3 Security**
   - Use IAM roles when possible
   - Enable S3 bucket encryption
   - Implement proper bucket policies

## Configuration

### S3 Configuration
```yaml
s3_bucket: "your-bucket"
region: "us-west-2"
aws_access_key: "YOUR_ACCESS_KEY"
aws_secret_key: "YOUR_SECRET_KEY"
s3_endpoint: "https://s3.amazonaws.com" # Optional
force_path_style: false # For S3-compatible storage
```

### Environment Variables
```bash
KUBECONFIG_S3_BUCKET="your-bucket"
KUBECONFIG_AWS_REGION="us-west-2"
KUBECONFIG_AWS_ACCESS_KEY="YOUR_ACCESS_KEY"
KUBECONFIG_AWS_SECRET_KEY="YOUR_SECRET_KEY"
```

## Troubleshooting

### Common Issues

1. **Token Creation Failed**
   ```bash
   # Verify cluster access
   kubconfig verify
   
   # Check minimum duration (10 minutes)
   kubconfig activate config.cfg --session 15m
   ```

2. **S3 Access Issues**
   ```bash
   # Verify S3 configuration
   kubconfig init
   
   # List bucket contents
   kubconfig list
   ```

3. **Session Cleanup**
   ```bash
   # Force cleanup
   kubconfig cleanup
   
   # Deactivate current session
   kubconfig deactivate
   ```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Setup

```bash
# Clone repository
git clone https://github.com/shahneil76/kubconfig-cli.git
cd kubconfig-cli

# Install dependencies
go mod download

# Build
go build -o kubconfig

# Run tests
go test ./...
```

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Support

- GitHub Issues: [Report a bug](https://github.com/shahneil76/kubconfig-cli/issues)
- Documentation: [Wiki](https://github.com/shahneil76/kubconfig-cli/wiki)
- Discussions: [GitHub Discussions](https://github.com/shahneil76/kubconfig-cli/discussions)

## Acknowledgments

- Kubernetes community
- AWS SDK for Go
- Cobra CLI framework

---

[![ko-fi](https://ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/T6T8NSVAN)
```

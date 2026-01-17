# Security Policy

## Supported Versions

The following versions of autospec are currently supported with security updates:

| Version | Supported          |
| ------- | ------------------ |
| 0.2.x   | :white_check_mark: |
| 0.1.x   | :white_check_mark: |
| < 0.1   | :x:                |

## Reporting a Vulnerability

We take security vulnerabilities seriously. If you discover a security issue, please report it responsibly.

### How to Report

**DO NOT** open a public GitHub issue for security vulnerabilities.

Instead, please report security vulnerabilities by emailing the maintainers directly or using GitHub's private vulnerability reporting feature:

1. Go to the [Security tab](https://github.com/ariel-frischer/autospec/security) of this repository
2. Click "Report a vulnerability"
3. Provide details about the vulnerability

### What to Include

When reporting a vulnerability, please include:

- **Description**: A clear description of the vulnerability
- **Impact**: What an attacker could achieve by exploiting this vulnerability
- **Steps to reproduce**: Detailed steps to reproduce the issue
- **Affected versions**: Which versions are affected
- **Possible fix**: If you have suggestions for how to fix the issue

### Response Timeline

- **Acknowledgment**: We will acknowledge receipt of your report within 48 hours
- **Initial assessment**: We will provide an initial assessment within 7 days
- **Resolution**: We aim to resolve critical vulnerabilities within 30 days

### Disclosure Policy

- We follow a coordinated disclosure process
- We will work with you to understand and resolve the issue
- Once fixed, we will publish a security advisory
- We will credit reporters (unless they prefer to remain anonymous)

## Security Best Practices

When using autospec:

### Configuration Security

- Store sensitive configuration in environment variables, not config files
- Do not commit `.autospec/config.json` if it contains sensitive data
- Use appropriate file permissions for config files

### API Key Management

- Never hardcode API keys in scripts or configuration
- Use environment variables for API keys (`ANTHROPIC_API_KEY`)
- Rotate API keys periodically

### Command Execution

- Be cautious when using `custom_claude_cmd` with untrusted input
- Validate spec names and paths before use
- Review generated commands before execution in production

### File System

- The tool writes to `~/.autospec/state/` for retry state
- Ensure appropriate permissions on state directory
- Review specs directory permissions

## Known Security Considerations

### Command Injection

The tool executes external commands (Claude CLI, SpecKit CLI). While we sanitize inputs, users should:

- Not pass untrusted input directly to commands
- Review custom command templates before use
- Be cautious with user-provided spec names in automated environments

### File Operations

The tool reads and writes files in the specs directory and state directory:

- Files are created with default permissions
- No special handling for symlinks (may follow symlinks)
- State files contain workflow metadata, not sensitive data

## Security Updates

Security updates will be released as patch versions. Subscribe to releases to be notified:

1. Go to the repository
2. Click "Watch" > "Custom" > Select "Releases"

## Contact

For security concerns that don't fit the vulnerability reporting process, contact the maintainers through the repository's discussion forum.

---

Thank you for helping keep autospec secure!

# Security Policy

## Supported Versions

We support security updates for the following versions:

| Version | Supported |
|---------|-----------|
| 1.x.x   | Yes       |
| < 1.0   | No        |

## Reporting a Vulnerability

If you discover a security vulnerability within this project, please do not open a public issue. Instead, report it directly by contacting the project maintainers:

- Email: `jezrealglobal@gmail.com`

We will acknowledge your report within 48 hours and work with you to analyze and resolve the issue.

## Local Data Security

Ropa-Sci stores all profile data locally in `data/players.json`.
- **Encryption:** The data is stored in plain-text JSON format. It is recommended to secure the filesystem of the host machine to prevent unauthorized local access.
- **Input Sanitization:** Usernames, names, and game moves are sanitized and validated to prevent injection or terminal escape sequence manipulation.
- **Concurrency Security:** File read/write operations on the JSON database are thread-safe and protected by a read-write sync mutex.

# Aptify — Self-hosted APT, simplified

![Version](https://img.shields.io/badge/version-v1.0.6-blue.svg)

Aptify is a self-hosted APT repository manager with a clean web UI. Host, sign, and serve your own `.deb` packages without any external infrastructure.

## Features

- **Web Dashboard** — Manage repositories and packages from a modern browser UI.
- **Automated Indexing** — Generates `Packages`, `Release`, and `InRelease` files automatically with GPG signatures on every upload.
- **Secure by Default** — Timing-safe authentication, strict path traversal protection, upload size limits, and GPG key export without token leakage via URL parameters.
- **Single Binary** — The entire backend and frontend ship as one self-contained Go binary.
- **Flexible Storage** — SQLite out of the box; MySQL supported for larger deployments.

## Quick Start

**Requirements:** Docker and Docker Compose.

1. Clone the repository.
2. Create a `.env` file:
   ```bash
   ADMIN_USERNAME=admin
   ADMIN_PASSWORD=mysecurepassword
   JWT_SECRET=your_super_secret_string
   KEY_NAME="My APT Repo"
   KEY_EMAIL="apt@mycompany.com"
   ```
3. Start the server:
   ```bash
   docker-compose up -d
   ```
4. Open `http://localhost:8080` in your browser.

## CLI Installation & Usage

Aptify provides a robust command-line client (`aptify-cli`) for easily pushing packages from your local machine or automating uploads in your CI/CD pipelines.

Install it in one line (macOS, Linux, and Windows via Git Bash):

```bash
curl -fsSL https://raw.githubusercontent.com/kernelcode0/aptify/main/install.sh | bash
```

**Usage Examples:**

```bash
# 1. Login locally (creates and saves an API key to ~/.config/aptify/config.json)
aptify-cli login http://localhost:8080

# 2. Push a package to your repository
aptify-cli push my-repo package_1.0.0_amd64.deb

# 3. Use it in a CI/CD script without logging in
aptify-cli push --server http://localhost:8080 --token YOUR_API_KEY my-repo *.deb
```

## Database

Aptify defaults to SQLite — no configuration needed. For larger deployments, switch to MySQL:

```bash
DB_TYPE=mysql
DB_DSN=user:password@tcp(127.0.0.1:3306)/aptify?parseTime=true
```

A commented-out MySQL service example is included in `docker-compose.yml`.

## Development

Requires Go 1.25+ and Node.js 20+.

```bash
# Build the frontend
cd web && npm run build

# Run the backend
go run ./cmd/server
```

Or use the Makefile targets:

```bash
make dev-backend   # runs the Go server with live reload
make dev-ui        # runs the Vite dev server
```

## Support

If you find this software useful and it saves you time or money, consider supporting its development!

<a href="https://www.buymeacoffee.com/kernelcode0" target="_blank"><img src="https://cdn.buymeacoffee.com/buttons/v2/default-yellow.png" alt="Buy Me A Coffee" style="height: 60px !important;width: 217px !important;" ></a>

### Why donate?

Your support directly funds:

- Server hosting and domain costs to keep the demo/project alive
- Late-night coding sessions and ongoing maintenance
- Keeping Aptify 100% free and open-source forever!

## Security

Authentication is backed by JWTs. Package filenames and upload paths are strictly validated. All file uploads are protected against DoS via size limits. The GPG signing key is exported through an authenticated API endpoint, never via URL parameters.

## Contact

For inquiries, support, or security reports, please contact: [aptify@kernelcode.de](mailto:aptify@kernelcode.de)

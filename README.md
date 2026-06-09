# Aptify

Aptify is a self-hosted APT repository manager with a beautiful web UI and robust security features. It allows you to easily manage your Debian packages, host custom APT repositories, and serve `.deb` packages securely.

## Features

- **Web Dashboard**: An intuitive and modern UI to manage your repositories and packages.
- **Automated Indexing**: Automatically generates `Packages`, `Release`, and `InRelease` files using GPG signatures.
- **Secure by Default**:
  - Timing attack mitigations on authentication routes.
  - Strict input validation to prevent path traversal attacks.
  - Secure memory footprint using size limits.
  - Secure GPG key export without exposing tokens via URL parameters.
- **Single Binary**: The entire backend and frontend are built into a single Go binary.
- **Lightweight**: Uses SQLite for fast and simple database management without complex dependencies.

## Installation & Deployment

We provide a simple Docker Compose setup.

1. Clone the repository.
2. Create a `.env` file with a strong `ADMIN_TOKEN`.
   ```bash
   ADMIN_TOKEN=your-secure-token
   KEY_NAME="My APT Repo"
   KEY_EMAIL="apt@mycompany.com"
   ```
3. Run with Docker Compose:
   ```bash
   docker-compose up -d
   ```
4. Access the web interface at `http://localhost:8080`.

## Database Configuration

Aptify uses SQLite by default, which requires zero configuration and stores data in the `DATA_DIR`.

If you prefer to use **MySQL** for larger deployments, you can configure it via environment variables:

```bash
DB_TYPE=mysql
DB_DSN=user:password@tcp(127.0.0.1:3306)/aptify?parseTime=true
```

You can find a commented-out example of a MySQL service in the `docker-compose.yml` file.

## Security

Aptify prioritizes security. The admin dashboard is protected via Bearer tokens. Packages and filenames are strictly validated, and file uploads are protected against DoS attacks via upload size limits.

## Development

Requires Go 1.25+ and Node.js 20+.

```bash
# Build the UI
cd web && npm run build

# Run the backend
go run ./cmd/server
```

You can also run `make dev-backend` and `make dev-ui` for a better development experience.

# Contributing to Aptify

First off, thanks for taking the time to contribute!

The following is a set of guidelines for contributing to Aptify.

## Development Environment Setup

Aptify uses Go for the backend and React (TypeScript) + Vite for the frontend.

### Prerequisites
- Go 1.25 or higher
- Node.js 20 or higher
- Make

### Running Locally

We have provided a `Makefile` to make development easy.

1. **Start the Backend:**
   In a terminal, run:
   ```bash
   make dev-backend
   ```
   This will start the Go API server at `http://localhost:8080`.

2. **Start the Frontend:**
   In a second terminal, run:
   ```bash
   make dev-ui
   ```
   This will start the Vite dev server at `http://localhost:5173`, proxying API requests to the backend.

## Submitting Pull Requests

1. Fork the repo and create your branch from `main`.
2. If you've added code that should be tested, add tests.
3. Ensure your Go code passes `go fmt` and `gosec`.
4. Ensure your frontend code builds cleanly (`npm run build`).
5. Issue that pull request!

## Code of Conduct

Please be respectful to other contributors. We strive to maintain a welcoming and inclusive community.

# Contributing to memlane (TodoMyDay)

Thank you for your interest in contributing to memlane! This document provides guidelines and instructions for contributing to the project.

## Code of Conduct

By participating in this project, you agree to maintain a respectful and inclusive environment for all contributors.

## How to Contribute

### Reporting Bugs

1. Check if the bug has already been reported in [Issues](https://github.com/entrefix/mrbrain/issues)
2. If not, create a new issue with:
   - A clear, descriptive title
   - Steps to reproduce the bug
   - Expected vs. actual behavior
   - Environment details (OS, Node.js version, Go version, etc.)
   - Screenshots if applicable

### Suggesting Features

1. Check if the feature has already been suggested
2. Open an issue with:
   - A clear description of the feature
   - Use cases and benefits
   - Any design considerations

### Pull Requests

1. **Fork the repository** and create a new branch from `main`
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes**
   - Follow the existing code style
   - Add comments for complex logic
   - Update documentation if needed
   - Add tests if applicable

3. **Test your changes**
   - Test locally with both frontend and backend
   - Ensure no linter errors
   - Verify the feature works as expected

4. **Commit your changes**
   - Use clear, descriptive commit messages
   - Follow conventional commit format when possible:
     - `feat: add new feature`
     - `fix: resolve bug`
     - `docs: update documentation`
     - `refactor: restructure code`
     - `test: add tests`

5. **Push and create a Pull Request**
   - Push to your fork
   - Create a PR with a clear description
   - Reference any related issues
   - Wait for review and address feedback

## Development Setup

### Prerequisites

- **Go 1.22+** for backend development
- **Node.js 18+** and npm for frontend development
- **Docker** (optional, for containerized development)
- **SQLite** (included with Go driver)

### Local Development

1. **Clone the repository**
   ```bash
   git clone https://github.com/entrefix/mrbrain.git
   cd mrbrain
   ```

2. **Set up environment variables**
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

3. **Backend Setup**
   ```bash
   cd backend
   go mod download
   go run ./cmd/server
   ```

4. **Frontend Setup**
   ```bash
   cd frontend
   npm install
   npm run dev
   ```

5. **Or use Docker**
   ```bash
   docker-compose up --build
   ```

## Code Style Guidelines

### Go (Backend)

- Follow [Effective Go](https://go.dev/doc/effective_go) guidelines
- Use `gofmt` to format code
- Keep functions focused and small
- Add comments for exported functions and types
- Handle errors explicitly (don't ignore them)

### TypeScript/React (Frontend)

- Use TypeScript for type safety
- Follow React best practices (hooks, functional components)
- Use Tailwind CSS for styling
- Keep components small and focused
- Use meaningful variable and function names

### General

- Write clear, self-documenting code
- Add comments for complex logic
- Keep functions under 50 lines when possible
- Use meaningful commit messages

## Project Structure

```
mrbrain/
├── backend/          # Go backend API
│   ├── cmd/         # Application entry points
│   ├── internal/    # Internal packages
│   │   ├── handlers/    # HTTP handlers
│   │   ├── services/    # Business logic
│   │   ├── repository/  # Data access layer
│   │   └── models/      # Domain models
│   └── go.mod
├── frontend/         # React frontend
│   ├── src/
│   │   ├── api/     # API client
│   │   ├── components/  # React components
│   │   ├── pages/   # Page components
│   │   └── contexts/    # React contexts
│   └── package.json
├── docs/            # Documentation
└── data/            # Database and vector storage (gitignored)
```

## Testing

Currently, the project doesn't have a comprehensive test suite. When adding tests:

- **Backend**: Use Go's built-in `testing` package
- **Frontend**: Use React Testing Library and Jest
- Write tests for new features and bug fixes
- Aim for good coverage of critical paths

## Documentation

- Update README.md if you add new features or change setup
- Add JSDoc/GoDoc comments for new functions
- Update API documentation if endpoints change
- Keep CONTRIBUTING.md up to date

## Questions?

- Open an issue for questions or discussions
- Check existing issues and PRs for similar questions
- Be patient and respectful in all interactions

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

Thank you for contributing to memlane! 🎉


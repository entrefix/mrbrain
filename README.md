# memlane - Memories App

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Contributions Welcome](https://img.shields.io/badge/contributions-welcome-brightgreen.svg)](CONTRIBUTING.md)

A personal productivity application with AI-powered task management and a smart memories/notes system.

## Features

### Todos
- **Todo Management**: Create, edit, delete, and reorder todos with drag-and-drop
- **Groups/Categories**: Organize todos into color-coded groups
- **Priority Levels**: Mark todos as low, medium, or high priority
- **AI Summarization**: Automatically cleans up todo titles and extracts relevant tags

### Memories
- **Quick Capture**: Save notes, links, ideas instantly
- **AI Categorization**: Automatically categorizes memories (Websites, Food, Movies, Books, Ideas, Places, Products, People, Learnings, Quotes)
- **URL Scraping**: Automatically fetches and summarizes linked content
- **Auto Web Search**: Detects search intent ("search about X", "what is Y") and fetches relevant information via SearXNG
- **Weekly Digest**: AI-generated summary of your week's memories
- **Convert to Todo**: Transform any memory into an actionable todo

### RAG & Search
- **Semantic Search**: Vector-based similarity search across todos and memories using embeddings
- **Hybrid Search**: Combines vector and keyword search with Reciprocal Rank Fusion (RRF) algorithm
- **Q&A System**: Ask questions about your data and get AI-generated answers with source attribution
- **Auto-Indexing**: Automatically indexes todos and memories for instant searchability
- **Chat Interface**: Interactive chat UI with Ask mode (Q&A) and Search mode (retrieval)
- **Full-Text Search**: SQLite FTS5 with Porter stemming for keyword matching

### General
- **Dark Mode**: Full dark mode support
- **Multiple AI Providers**: Supports OpenAI, Anthropic, Google, and custom OpenAI-compatible APIs
- **JWT Authentication**: Secure authentication with httpOnly cookies

## Architecture

```
todomyday/
├── frontend/          # React + TypeScript + Vite
├── backend/           # Go + Gin + SQLite
├── data/              # SQLite database + vector storage (created on first run)
├── docker-compose.yml
└── .env.example
```

## RAG System

TodoDay includes a sophisticated RAG (Retrieval-Augmented Generation) system for intelligent search and Q&A over your personal data.

### Overview

The RAG system enables you to:
- Search your todos and memories using natural language
- Ask questions and get AI-generated answers with source citations
- Combine semantic understanding with keyword matching for better results

### Components

**Vector Database**
- Uses `chromem-go` for efficient vector storage and similarity search
- Persistent storage at `./data/vectors` (configurable)
- Supports filtering by user, content type, and metadata

**Full-Text Search**
- SQLite FTS5 virtual tables for keyword matching
- Porter stemming for better word matching
- Auto-synced with main tables via triggers
- Highlighted snippets in search results

**Embedding Service**
- NVIDIA NIM embedding API for vector search
- Default model: `nvidia/nv-embedqa-e5-v5` (1024 dimensions)
- Rate-limited to 40 requests per minute
- Separate query/passage embeddings for optimal retrieval
- Text sanitization for special characters and unicode

**Hybrid Search Algorithm**
- Combines vector similarity and keyword search results
- Reciprocal Rank Fusion (RRF) for optimal ranking
- Configurable vector weight (default: 70% vector, 30% keyword)
- Returns match type: vector, keyword, or hybrid

### Search Modes

**Semantic/Vector Search**
- Finds conceptually similar content even with different wording
- Example: "exercise ideas" matches memories about "workout plans" or "fitness goals"
- Uses embeddings to understand meaning, not just keywords

**Keyword Search**
- Traditional full-text search with stemming
- Fast and precise for exact terms
- Good for searching specific names, dates, or technical terms

**Hybrid (Default)**
- Best of both worlds: semantic understanding + keyword precision
- Automatically combines and ranks results from both approaches
- Adapts to different query types

### Q&A System

**How It Works:**
1. Your question is embedded and used to search relevant documents
2. Top matches are retrieved from todos and memories
3. Context is built from search results
4. AI generates an answer using your configured provider
5. Sources are returned with relevance scores

**Example:**
- Question: "What movies did I want to watch?"
- System retrieves relevant memories about movies
- AI generates: "Based on your memories, you wanted to watch..."
- Shows source memories with match scores

**Features:**
- Configurable result limits (default: 5 sources)
- Content type filtering (search only todos, only memories, or both)
- Uses your preferred AI provider (OpenAI, Anthropic, Google, custom)
- Graceful degradation if RAG is disabled

## Quick Start

### 1. Clone and Configure

```bash
# Copy environment file
cp .env.example .env

# Edit .env with your settings
# Required: JWT_SECRET (min 32 characters)
# Optional: OPENAI_* for AI features
```

### 2. Run with Docker

```bash
docker-compose up --build
```

- Frontend: http://localhost:3111
- Backend API: http://localhost:8099

### 3. Development (without Docker)

**Backend:**
```bash
cd backend
go mod download
go run ./cmd/server
```

**Frontend:**
```bash
cd frontend
npm install
npm run dev
```

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `JWT_SECRET` | Yes | - | Secret key for JWT signing (min 32 chars) |
| `JWT_EXPIRATION` | No | `24h` | JWT token expiration |
| `DATABASE_PATH` | No | `./data/todomyday.db` | SQLite database path |
| `ENCRYPTION_KEY` | No | dev key | Key for encrypting API keys (32 chars for production) |
| `OPENAI_BASE_URL` | No | - | Default OpenAI API base URL |
| `OPENAI_API_KEY` | No | - | Default OpenAI API key |
| `OPENAI_MODEL` | No | `gpt-3.5-turbo` | Default model for AI features |
| `VECTOR_DB_PATH` | No | `./data/vectors` | Path for vector database storage |
| `RAG_ENABLED` | No | `true` | Enable/disable RAG features |
| `SEARXNG_URLS` | No | - | Comma-separated SearXNG instance URLs for web search |
| `ALLOWED_ORIGINS` | No | `http://localhost:3111` | CORS allowed origins |
| `VITE_API_URL` | No | `http://localhost:8099` | Backend API URL for frontend |

### NIM Embedding Settings (Required for RAG)

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `NIM_API_KEY` | Yes* | - | NVIDIA NIM API key (get from https://build.nvidia.com) |
| `NIM_BASE_URL` | No | `https://integrate.api.nvidia.com/v1` | NIM API base URL |
| `NIM_MODEL` | No | `nvidia/nv-embedqa-e5-v5` | NIM embedding model |
| `NIM_RPM_LIMIT` | No | `40` | Rate limit (requests per minute) |
| `NIM_EMBEDDING_DIM` | No | `1024` | Embedding dimension |

*Required if `RAG_ENABLED=true`

## API Endpoints

### Auth
- `POST /api/auth/register` - Create new account
- `POST /api/auth/login` - Login
- `POST /api/auth/logout` - Logout
- `GET /api/auth/me` - Get current user

### Todos
- `GET /api/todos` - List all todos
- `POST /api/todos` - Create todo (with AI processing if configured)
- `PUT /api/todos/:id` - Update todo
- `DELETE /api/todos/:id` - Delete todo
- `PUT /api/todos/reorder` - Reorder todos

### Groups
- `GET /api/groups` - List all groups (user's + defaults)
- `POST /api/groups` - Create group
- `PUT /api/groups/:id` - Update group
- `DELETE /api/groups/:id` - Delete group

### Memories
- `GET /api/memories` - List all memories (with pagination)
- `POST /api/memories` - Create memory (with AI categorization + URL/search processing)
- `GET /api/memories/:id` - Get single memory
- `PUT /api/memories/:id` - Update memory
- `DELETE /api/memories/:id` - Delete memory
- `POST /api/memories/search` - Full-text search memories
- `GET /api/memories/categories` - Get category list with counts
- `GET /api/memories/stats` - Get memory statistics
- `GET /api/memories/digest` - Get/generate weekly digest
- `POST /api/memories/:id/convert-to-todo` - Convert memory to todo
- `POST /api/memories/web-search` - Manual web search

### AI Providers
- `GET /api/ai-providers` - List user's AI providers
- `POST /api/ai-providers` - Add AI provider
- `PUT /api/ai-providers/:id` - Update provider
- `DELETE /api/ai-providers/:id` - Delete provider
- `POST /api/ai-providers/:id/test` - Test provider connection
- `GET /api/ai-providers/:id/models` - Fetch available models

### RAG & Search
- `POST /api/rag/search` - Hybrid semantic + keyword search across todos and memories
- `POST /api/rag/ask` - Ask questions and get AI-generated answers with sources
- `POST /api/rag/index` - Manually trigger indexing for user's todos and memories
- `GET /api/rag/stats` - Get index statistics and RAG configuration status

## Tech Stack

**Frontend:**
- React 18 + TypeScript
- Vite
- Tailwind CSS
- Framer Motion
- React Beautiful DnD
- Axios

**Backend:**
- Go 1.22
- Gin Web Framework
- SQLite (modernc.org/sqlite - pure Go)
- SQLite FTS5 (Full-Text Search)
- chromem-go (Vector Database)
- NVIDIA NIM Embeddings API
- JWT Authentication
- bcrypt Password Hashing

## Contributing

Contributions are welcome! Please read our [Contributing Guide](CONTRIBUTING.md) and [Code of Conduct](CODE_OF_CONDUCT.md) for details on our community standards and the process for submitting pull requests.

## Security

If you discover a security vulnerability, please follow our [Security Policy](SECURITY.md) and report it privately.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built with [React](https://react.dev/), [Go](https://go.dev/), and [SQLite](https://www.sqlite.org/)
- Vector embeddings powered by [NVIDIA NIM](https://build.nvidia.com)
- UI components styled with [Tailwind CSS](https://tailwindcss.com/)

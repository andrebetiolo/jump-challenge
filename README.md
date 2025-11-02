# Jump Challenge - Intelligent Email Organization App

This is an intelligent email organization application that uses AI to classify and summarize emails, integrating with Gmail via OAuth.

## Features

- OAuth login with Google
- Create, read, update, and delete email categories
- Automatic email classification using AI
- Email summarization using AI
- Gmail integration (read, archive, mark as read)
- Bulk email actions
- Session-based authentication

## Architecture

The application follows a modular architecture with clear separation of concerns:

- **Handlers**: Handle HTTP requests and responses
- **Services**: Implement business logic
- **Repositories**: Handle data persistence
- **Models**: Define data structures
- **Gmail**: Interact with Gmail API
- **AI**: Interact with AI services

## Prerequisites

- Go 1.21+
- Google Cloud Project with Gmail API enabled
- AI service API key (e.g., OpenAI, DeepSeek)

## Setup

1. Clone the repository
2. Copy `.env.example` to `.env` and fill in your credentials:

```bash
cp .env.example .env
```

3. Install dependencies:

```bash
go mod download
```

4. Run the application:

```bash
go run cmd/server/main.go
```

## Environment Variables

- `PORT`: Port to run the server on (default: 8080)
- `BASE_URL`: Base URL of the application
- `GOOGLE_CLIENT_ID`: Google OAuth client ID
- `GOOGLE_CLIENT_SECRET`: Google OAuth client secret
- `SESSION_SECRET`: Secret for session encryption
- `DATABASE_URL`: Database connection string (optional for in-memory)
- `AI_API_KEY`: API key for AI service
- `AI_PROVIDER`: AI provider (default: openai, can be deepseek)
- `ENV`: Environment (development/production)

## API Endpoints

### Authentication
- `GET /auth/google` - Initiate Google OAuth
- `GET /auth/google/callback` - OAuth callback
- `POST /auth/logout` - Logout

### Categories
- `POST /categories` - Create category
- `GET /categories` - List categories
- `GET /categories/:id` - Get category
- `PUT /categories/:id` - Update category
- `DELETE /categories/:id` - Delete category

### Emails
- `GET /emails` - List user's emails
- `GET /emails/category/:id` - Get emails by category
- `POST /emails/sync` - Sync emails from Gmail
- `POST /emails/bulk-action` - Perform bulk action on emails

## Development

The application uses in-memory storage by default. To run tests:

```bash
go test ./...
```

## Technologies Used

- Go 1.21+
- Echo web framework
- Goth OAuth library
- Gmail API
- OpenAI API, DeepSeek API (or other AI services)
- PostgreSQL (optional)
- Gorilla sessions

## Configuration

The application follows the 12-factor app methodology with configuration via environment variables. It supports both development and production environments.
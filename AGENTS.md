# Jump Challenge - Intelligent Email Organization App

## Project Overview

This is an intelligent email organization application that uses AI to classify and summarize emails, integrating with Gmail via OAuth.

## Architecture

The application follows a modular architecture with clear separation of concerns:

- **Handlers**: Handle HTTP requests and responses
- **Services**: Implement business logic
- **Repositories**: Handle data persistence
- **Models**: Define data structures
- **Gmail**: Interact with Gmail API
- **AI**: Interact with AI services

## Current Implementation

### Models
- **User**: Stores Google ID, email, name, access tokens, etc.
- **Category**: Email classification categories with name and description
- **Email**: Email objects with subject, body, summary, classification, etc.

### Repository Layer
- **Interfaces**: UserRepository, CategoryRepository, EmailRepository
- **Memory Implementation**: In-memory storage for development/testing
- **PostgreSQL Implementation**: Production-ready database implementation

### Service Layer
- **Auth Service**: Handles user authentication and OAuth
- **Category Service**: Full CRUD operations for categories
- **Email Service**: Gmail synchronization, AI processing, bulk actions

### API Clients
- **Gmail Client**: Interface to interact with Gmail API
- **AI Client**: Interface to interact with AI services for classification and summarization

### Handlers
- **Auth Handler**: Google OAuth authentication endpoints
- **Category Handler**: CRUD operations for categories
- **Email Handler**: Email synchronization and bulk actions

### Middleware
- **Auth Middleware**: Protects routes requiring authentication

### Configuration & Utilities
- **Config**: Environment variable management
- **Logger**: Application logging
- **Router**: Echo router setup

## Endpoints

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

## Key Features

1. **OAuth Integration**: Login with Google using goth library
2. **Email Classification**: Automatic classification of emails using AI
3. **Email Summarization**: AI-generated summaries of email content
4. **Gmail Integration**: Read, archive, mark as read operations
5. **Bulk Operations**: Mass actions on selected emails
6. **Category Management**: Create, update, delete email categories

## Technology Stack

- **Go 1.21+**
- **Echo**: Web framework
- **Goth**: OAuth library
- **Gmail API**: Google's Gmail API
- **OpenAI/DeepSeek**: AI service integration
- **Gorilla Sessions**: Session management
- **PostgreSQL**: Production database (with in-memory for testing)

## Environment Variables

- `PORT`: Port to run the server on
- `BASE_URL`: Base URL of the application
- `GOOGLE_CLIENT_ID`: Google OAuth client ID
- `GOOGLE_CLIENT_SECRET`: Google OAuth client secret
- `SESSION_SECRET`: Secret for session encryption
- `DATABASE_URL`: Database connection string
- `AI_API_KEY`: API key for AI service
- `AI_PROVIDER`: AI provider (default: gemini, can be deepseek)
- `ENV`: Environment (development/production)

## Testing

Unit tests for services with mocks are included in the `/tests/` directory.

## Development Notes

### Adding New Features

1. **New Endpoints**: Create new handlers in `/internal/handler/`, register routes in `/internal/router/router.go`
2. **New Services**: Add service interfaces in `/internal/service/interface.go`, implement in `/internal/service/`
3. **New Models**: Add to `/internal/model/`
4. **New Repository Methods**: Extend interfaces in `/internal/repository/interface.go`, implement in both memory and postgres packages

### Authentication Flow
1. User clicks "Login with Google"
2. Goth redirects to Google's OAuth
3. Google redirects back to callback endpoint
4. User is created/fetched from database
5. Session is created
6. User is redirected to dashboard

### Email Processing Flow
1. User triggers email sync (manual or automated)
2. App fetches new unread emails from Gmail
3. For each email:
   - AI classifies email into a category
   - AI generates summary
   - Email is saved to database with classification
   - Email is archived in Gmail
4. Processed emails are available in categorized views

## Future Enhancements

### Potential Features
- Scheduled email synchronization
- Advanced filtering options
- Email templates and responses
- User preferences and settings
- Webhook support for real-time updates
- Multiple email account support
- Advanced AI model fine-tuning
- Email threading/conversation view

### Improvements
- Enhanced error handling and retry logic
- Better logging and monitoring
- Performance optimizations
- Caching layer for frequently accessed data
- More comprehensive test coverage
- Rate limiting and security enhancements
- Admin panel for user management
- Export functionality for reports
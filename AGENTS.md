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
2. App fetches new emails from Gmail (using SyncEmails function)
3. For each email:
   - AI classifies email into a category
   - AI generates summary
   - Email is saved to database with classification
   - Email is archived in Gmail
4. Processed emails are available in categorized views

## Implementation Guidelines for Faster Development

### Key Principles
1. **Interface-First Development**: Always update interfaces in `/internal/service/interface.go` first, then implementations
2. **Dependency Injection**: Services depend on interfaces, not concrete implementations
3. **Error Handling**: Wrap errors with context using `fmt.Errorf("context: %w", err)`
4. **Logging**: Use the provided logger consistently throughout the application
5. **Context Propagation**: Always pass context through function calls

### Critical Files to Check When Making Changes
1. **`internal/service/interface.go`**: Contains all service interfaces - update first when adding/changing methods
2. **`internal/gmail/gmail_client.go`**: Gmail API interactions - update when changing email sync logic
3. **`internal/service/email_service.go`**: Email business logic - update when changing sync behavior
4. **`internal/handler/email_handler.go`**: HTTP handler for email endpoints - update when changing API parameters
5. **`tests/service_test.go`**: Update tests when changing service method signatures
6. **`internal/gmail/mock_client.go`**: Update mock implementations when changing Gmail client interface

### Common Patterns When Adding Parameters to Functions

When adding new parameters to existing functions (like in the recent email sync update):

1. **Update Interface First**: Modify the interface in `internal/service/interface.go`
2. **Update All Implementations**: Update every struct that implements the interface (including mocks)
3. **Update Service Layer**: Modify the service methods to accept and pass the new parameters
4. **Update Handler**: Modify the handler to extract parameters from request and pass to service
5. **Update Tests**: Update test files with the new method signatures
6. **Maintain Backward Compatibility**: Default to sensible values when parameters are not provided

### Method Naming Conventions
- Use descriptive names that reflect actual functionality (recently changed `ListUnreadEmails` to `SyncEmails` since it fetches all emails, not just unread ones)
- Be consistent with domain terminology (e.g., use terms like "sync", "archive", "classify" etc.)

### Testing Strategy
- Update mock clients in `internal/gmail/mock_client.go` to match new function signatures
- Add test cases for new functionality in `tests/service_test.go`
- Run `go test ./...` to ensure all tests pass after changes
- Run `go build` to ensure compilation succeeds

### API Parameter Handling
- Use query parameters for optional parameters in GET requests (e.g., `?max_results=20&after_email_id=12345`)
- Use request body for required parameters in POST requests
- Always validate and sanitize input parameters
- Provide sensible defaults when parameters are missing

### Gmail Integration Specifics
- When modifying email fetching logic, ensure both the Gmail client and its mock implementations are updated
- The Gmail sync function now accepts `maxResults` (int64) and `afterEmailID` (string) parameters
- The function can fetch the last X emails or fetch emails after a specific email ID
- Maintain compatibility with the `MAX_FETCH_EMAILS` environment variable as fallback

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
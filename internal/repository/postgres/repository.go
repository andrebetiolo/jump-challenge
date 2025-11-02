package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"jump-challenge/internal/model"

	_ "github.com/lib/pq"
)

type PostgresUserRepository struct {
	db *sql.DB
}

func NewPostgresUserRepository(db *sql.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

func (r *PostgresUserRepository) Create(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO users (id, google_id, email, name, access_token, refresh_token, token_expiry, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (google_id) DO UPDATE SET
			email = EXCLUDED.email,
			name = EXCLUDED.name,
			access_token = EXCLUDED.access_token,
			refresh_token = EXCLUDED.refresh_token,
			token_expiry = EXCLUDED.token_expiry,
			updated_at = NOW()`
	_, err := r.db.ExecContext(ctx, query,
		user.ID, user.GoogleID, user.Email, user.Name,
		user.AccessToken, user.RefreshToken, user.TokenExpiry,
		user.CreatedAt, user.UpdatedAt)
	return err
}

func (r *PostgresUserRepository) FindByID(ctx context.Context, id string) (*model.User, error) {
	query := `SELECT id, google_id, email, name, access_token, refresh_token, token_expiry, created_at, updated_at FROM users WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, id)

	user := &model.User{}
	err := row.Scan(
		&user.ID, &user.GoogleID, &user.Email, &user.Name,
		&user.AccessToken, &user.RefreshToken, &user.TokenExpiry,
		&user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return user, nil
}

func (r *PostgresUserRepository) FindByGoogleID(ctx context.Context, googleID string) (*model.User, error) {
	query := `SELECT id, google_id, email, name, access_token, refresh_token, token_expiry, created_at, updated_at FROM users WHERE google_id = $1`
	row := r.db.QueryRowContext(ctx, query, googleID)

	user := &model.User{}
	err := row.Scan(
		&user.ID, &user.GoogleID, &user.Email, &user.Name,
		&user.AccessToken, &user.RefreshToken, &user.TokenExpiry,
		&user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return user, nil
}

func (r *PostgresUserRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `SELECT id, google_id, email, name, access_token, refresh_token, token_expiry, created_at, updated_at FROM users WHERE email = $1`
	row := r.db.QueryRowContext(ctx, query, email)

	user := &model.User{}
	err := row.Scan(
		&user.ID, &user.GoogleID, &user.Email, &user.Name,
		&user.AccessToken, &user.RefreshToken, &user.TokenExpiry,
		&user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return user, nil
}

func (r *PostgresUserRepository) Update(ctx context.Context, user *model.User) error {
	query := `
		UPDATE users SET google_id=$1, email=$2, name=$3, access_token=$4, 
		refresh_token=$5, token_expiry=$6, updated_at=NOW() WHERE id=$7`
	_, err := r.db.ExecContext(ctx, query,
		user.GoogleID, user.Email, user.Name,
		user.AccessToken, user.RefreshToken, user.TokenExpiry,
		user.ID)
	return err
}

func (r *PostgresUserRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// Postgres Category repository implementation
type PostgresCategoryRepository struct {
	db *sql.DB
}

func NewPostgresCategoryRepository(db *sql.DB) *PostgresCategoryRepository {
	return &PostgresCategoryRepository{db: db}
}

func (r *PostgresCategoryRepository) Create(ctx context.Context, category *model.Category) error {
	query := `
		INSERT INTO categories (id, name, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			updated_at = NOW()`
	_, err := r.db.ExecContext(ctx, query,
		category.ID, category.Name, category.Description,
		category.CreatedAt, category.UpdatedAt)
	return err
}

func (r *PostgresCategoryRepository) FindByID(ctx context.Context, id string) (*model.Category, error) {
	query := `SELECT id, name, description, created_at, updated_at FROM categories WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, id)

	category := &model.Category{}
	err := row.Scan(
		&category.ID, &category.Name, &category.Description,
		&category.CreatedAt, &category.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("category not found")
		}
		return nil, err
	}
	return category, nil
}

func (r *PostgresCategoryRepository) FindAll(ctx context.Context) ([]*model.Category, error) {
	query := `SELECT id, name, description, created_at, updated_at FROM categories`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []*model.Category
	for rows.Next() {
		category := &model.Category{}
		err := rows.Scan(
			&category.ID, &category.Name, &category.Description,
			&category.CreatedAt, &category.UpdatedAt)
		if err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}

	return categories, rows.Err()
}

func (r *PostgresCategoryRepository) Update(ctx context.Context, category *model.Category) error {
	query := `
		UPDATE categories SET name=$1, description=$2, updated_at=NOW() WHERE id=$3`
	_, err := r.db.ExecContext(ctx, query,
		category.Name, category.Description, category.ID)
	return err
}

func (r *PostgresCategoryRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM categories WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// Postgres Email repository implementation
type PostgresEmailRepository struct {
	db *sql.DB
}

func NewPostgresEmailRepository(db *sql.DB) *PostgresEmailRepository {
	return &PostgresEmailRepository{db: db}
}

func (r *PostgresEmailRepository) Create(ctx context.Context, email *model.Email) error {
	query := `
		INSERT INTO emails (id, user_id, gmail_id, from_email, subject, body, summary, category_id, received_at, archived, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (gmail_id) DO UPDATE SET
			user_id = EXCLUDED.user_id,
			from_email = EXCLUDED.from_email,
			subject = EXCLUDED.subject,
			body = EXCLUDED.body,
			summary = EXCLUDED.summary,
			category_id = EXCLUDED.category_id,
			received_at = EXCLUDED.received_at,
			archived = EXCLUDED.archived,
			updated_at = NOW()`
	_, err := r.db.ExecContext(ctx, query,
		email.ID, email.UserID, email.GmailID, email.From, email.Subject, email.Body,
		email.Summary, email.CategoryID, email.ReceivedAt, email.Archived,
		email.CreatedAt, email.UpdatedAt)
	return err
}

func (r *PostgresEmailRepository) FindByID(ctx context.Context, id string) (*model.Email, error) {
	query := `SELECT id, user_id, gmail_id, from_email, subject, body, summary, category_id, received_at, archived, created_at, updated_at FROM emails WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, id)

	email := &model.Email{}
	err := row.Scan(
		&email.ID, &email.UserID, &email.GmailID, &email.From, &email.Subject, &email.Body,
		&email.Summary, &email.CategoryID, &email.ReceivedAt, &email.Archived,
		&email.CreatedAt, &email.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("email not found")
		}
		return nil, err
	}
	return email, nil
}

func (r *PostgresEmailRepository) FindByUserID(ctx context.Context, userID string) ([]*model.Email, error) {
	query := `SELECT id, user_id, gmail_id, from_email, subject, body, summary, category_id, received_at, archived, created_at, updated_at FROM emails WHERE user_id = $1`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var emails []*model.Email
	for rows.Next() {
		email := &model.Email{}
		err := rows.Scan(
			&email.ID, &email.UserID, &email.GmailID, &email.From, &email.Subject, &email.Body,
			&email.Summary, &email.CategoryID, &email.ReceivedAt, &email.Archived,
			&email.CreatedAt, &email.UpdatedAt)
		if err != nil {
			return nil, err
		}
		emails = append(emails, email)
	}

	return emails, nil
}

func (r *PostgresEmailRepository) FindByCategoryID(ctx context.Context, categoryID string) ([]*model.Email, error) {
	query := `SELECT id, user_id, gmail_id, from_email, subject, body, summary, category_id, received_at, archived, created_at, updated_at FROM emails WHERE category_id = $1`
	rows, err := r.db.QueryContext(ctx, query, categoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var emails []*model.Email
	for rows.Next() {
		email := &model.Email{}
		err := rows.Scan(
			&email.ID, &email.UserID, &email.GmailID, &email.From, &email.Subject, &email.Body,
			&email.Summary, &email.CategoryID, &email.ReceivedAt, &email.Archived,
			&email.CreatedAt, &email.UpdatedAt)
		if err != nil {
			return nil, err
		}
		emails = append(emails, email)
	}

	return emails, nil
}

func (r *PostgresEmailRepository) Update(ctx context.Context, email *model.Email) error {
	query := `
		UPDATE emails SET from_email=$1, subject=$2, body=$3, summary=$4, category_id=$5, archived=$6, updated_at=NOW() WHERE id=$7`
	_, err := r.db.ExecContext(ctx, query,
		email.From, email.Subject, email.Body, email.Summary, email.CategoryID, email.Archived,
		email.ID)
	return err
}

func (r *PostgresEmailRepository) FindByGmailID(ctx context.Context, userID, gmailID string) (*model.Email, error) {
	query := `SELECT id, user_id, gmail_id, from_email, subject, body, summary, category_id, received_at, archived, created_at, updated_at FROM emails WHERE user_id = $1 AND gmail_id = $2`
	row := r.db.QueryRowContext(ctx, query, userID, gmailID)

	email := &model.Email{}
	err := row.Scan(
		&email.ID, &email.UserID, &email.GmailID, &email.From, &email.Subject, &email.Body,
		&email.Summary, &email.CategoryID, &email.ReceivedAt, &email.Archived,
		&email.CreatedAt, &email.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("email not found")
		}
		return nil, err
	}
	return email, nil
}

func (r *PostgresEmailRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM emails WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// InitializeDatabase creates the necessary tables
func InitializeDatabase(db *sql.DB) error {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id VARCHAR(255) PRIMARY KEY,
			google_id VARCHAR(255) UNIQUE NOT NULL,
			email VARCHAR(255) NOT NULL,
			name VARCHAR(255) NOT NULL,
			access_token TEXT,
			refresh_token TEXT,
			token_expiry TIMESTAMP,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS categories (
			id VARCHAR(255) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS emails (
			id VARCHAR(255) PRIMARY KEY,
			user_id VARCHAR(255) NOT NULL,
			gmail_id VARCHAR(255) UNIQUE NOT NULL,
			from_email TEXT,
			subject TEXT NOT NULL,
			body TEXT,
			summary TEXT,
			category_id VARCHAR(255),
			received_at TIMESTAMP NOT NULL,
			archived BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)`,
	}

	for _, table := range tables {
		_, err := db.Exec(table)
		if err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	return nil
}

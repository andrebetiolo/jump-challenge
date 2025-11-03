package memory

import (
	"context"
	"errors"
	"sync"

	"jump-challenge/internal/model"
)

type InMemoryUserRepository struct {
	users map[string]*model.User
	mutex sync.RWMutex
}

func NewInMemoryUserRepository() *InMemoryUserRepository {
	return &InMemoryUserRepository{
		users: make(map[string]*model.User),
	}
}

func (r *InMemoryUserRepository) Create(ctx context.Context, user *model.User) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	r.users[user.ID] = user
	return nil
}

func (r *InMemoryUserRepository) FindByID(ctx context.Context, id string) (*model.User, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	user, exists := r.users[id]
	if !exists {
		return nil, errors.New("user not found")
	}
	return user, nil
}

func (r *InMemoryUserRepository) FindByGoogleID(ctx context.Context, googleID string) (*model.User, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	for _, user := range r.users {
		if user.GoogleID == googleID {
			return user, nil
		}
	}
	return nil, errors.New("user not found")
}

func (r *InMemoryUserRepository) Update(ctx context.Context, user *model.User) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	_, exists := r.users[user.ID]
	if !exists {
		return errors.New("user not found")
	}
	r.users[user.ID] = user
	return nil
}

func (r *InMemoryUserRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	for _, user := range r.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, errors.New("user not found")
}

func (r *InMemoryUserRepository) Delete(ctx context.Context, id string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	delete(r.users, id)
	return nil
}

func (r *InMemoryUserRepository) FindAll(ctx context.Context) ([]*model.User, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	var users []*model.User
	for _, user := range r.users {
		users = append(users, user)
	}
	return users, nil
}

// GetAllUsers returns all users (needed for the Gmail client to find users by email)
func (r *InMemoryUserRepository) GetAllUsers() []*model.User {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	var users []*model.User
	for _, user := range r.users {
		users = append(users, user)
	}
	return users
}

// Category repository implementation
type InMemoryCategoryRepository struct {
	categories map[string]*model.Category
	mutex      sync.RWMutex
}

func NewInMemoryCategoryRepository() *InMemoryCategoryRepository {
	return &InMemoryCategoryRepository{
		categories: make(map[string]*model.Category),
	}
}

func (r *InMemoryCategoryRepository) Create(ctx context.Context, category *model.Category) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	r.categories[category.ID] = category
	return nil
}

func (r *InMemoryCategoryRepository) FindByID(ctx context.Context, id string) (*model.Category, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	category, exists := r.categories[id]
	if !exists {
		return nil, errors.New("category not found")
	}
	return category, nil
}

func (r *InMemoryCategoryRepository) FindAll(ctx context.Context) ([]*model.Category, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	var result []*model.Category
	for _, category := range r.categories {
		result = append(result, category)
	}
	return result, nil
}

func (r *InMemoryCategoryRepository) Update(ctx context.Context, category *model.Category) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	_, exists := r.categories[category.ID]
	if !exists {
		return errors.New("category not found")
	}
	r.categories[category.ID] = category
	return nil
}

func (r *InMemoryCategoryRepository) Delete(ctx context.Context, id string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	delete(r.categories, id)
	return nil
}

// Email repository implementation
type InMemoryEmailRepository struct {
	emails map[string]*model.Email
	mutex  sync.RWMutex
}

func NewInMemoryEmailRepository() *InMemoryEmailRepository {
	return &InMemoryEmailRepository{
		emails: make(map[string]*model.Email),
	}
}

func (r *InMemoryEmailRepository) Create(ctx context.Context, email *model.Email) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	r.emails[email.ID] = email
	return nil
}

func (r *InMemoryEmailRepository) FindByID(ctx context.Context, id string) (*model.Email, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	email, exists := r.emails[id]
	if !exists {
		return nil, errors.New("email not found")
	}
	return email, nil
}

func (r *InMemoryEmailRepository) FindByUserID(ctx context.Context, userID string) ([]*model.Email, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	var result []*model.Email
	for _, email := range r.emails {
		if email.UserID == userID {
			result = append(result, email)
		}
	}
	return result, nil
}

func (r *InMemoryEmailRepository) FindByCategoryID(ctx context.Context, categoryID string) ([]*model.Email, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	var result []*model.Email
	for _, email := range r.emails {
		if email.CategoryID == categoryID {
			result = append(result, email)
		}
	}
	return result, nil
}

func (r *InMemoryEmailRepository) FindByGmailID(ctx context.Context, userID, gmailID string) (*model.Email, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	for _, email := range r.emails {
		if email.UserID == userID && email.GmailID == gmailID {
			return email, nil
		}
	}
	return nil, errors.New("email not found")
}

func (r *InMemoryEmailRepository) Update(ctx context.Context, email *model.Email) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	_, exists := r.emails[email.ID]
	if !exists {
		return errors.New("email not found")
	}
	r.emails[email.ID] = email
	return nil
}

func (r *InMemoryEmailRepository) Delete(ctx context.Context, id string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	delete(r.emails, id)
	return nil
}
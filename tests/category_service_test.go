package tests

import (
	"context"
	"testing"

	"jump-challenge/internal/logger"
	"jump-challenge/internal/repository/memory"
	"jump-challenge/internal/service"

	"github.com/stretchr/testify/assert"
)

func TestCategoryServiceCRUD(t *testing.T) {
	// Setup
	categoryRepo := memory.NewInMemoryCategoryRepository()
	appLogger := logger.New()

	// Create service
	categoryService := service.NewCategoryService(categoryRepo, appLogger)

	// Test Create
	category, err := categoryService.CreateCategory(context.Background(), "", "Work", "Work related emails")
	assert.NoError(t, err)
	assert.Equal(t, "Work", category.Name)
	assert.Equal(t, "Work related emails", category.Description)

	// Test Get by ID
	retrievedCategory, err := categoryService.GetCategory(context.Background(), category.ID)
	assert.NoError(t, err)
	assert.Equal(t, category.ID, retrievedCategory.ID)
	assert.Equal(t, "Work", retrievedCategory.Name)

	// Test Get all categories
	categories, err := categoryService.GetAllCategories(context.Background())
	assert.NoError(t, err)
	assert.Len(t, categories, 1)
	assert.Equal(t, "Work", categories[0].Name)

	// Test Update
	updatedCategory, err := categoryService.UpdateCategory(context.Background(), category.ID, "Updated Work", "Updated description")
	assert.NoError(t, err)
	assert.Equal(t, "Updated Work", updatedCategory.Name)
	assert.Equal(t, "Updated description", updatedCategory.Description)

	// Test Delete
	err = categoryService.DeleteCategory(context.Background(), category.ID)
	assert.NoError(t, err)

	// Verify deletion
	_, err = categoryService.GetCategory(context.Background(), category.ID)
	assert.Error(t, err)
}

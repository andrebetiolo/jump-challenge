package service

import (
	"context"
	"time"

	"jump-challenge/internal/logger"
	"jump-challenge/internal/model"
	"jump-challenge/internal/repository"
)

type categoryService struct {
	categoryRepo repository.CategoryRepository
	logger       *logger.Logger
}

func NewCategoryService(categoryRepo repository.CategoryRepository, logger *logger.Logger) CategoryService {
	return &categoryService{
		categoryRepo: categoryRepo,
		logger:       logger,
	}
}

func (s *categoryService) CreateCategory(ctx context.Context, userID, name, description string) (*model.Category, error) {
	category := model.NewCategory(name, description)
	if err := s.categoryRepo.Create(ctx, category); err != nil {
		s.logger.Error("Failed to create category:", err)
		return nil, err
	}
	s.logger.Info("Created category:", category.ID)
	return category, nil
}

func (s *categoryService) GetCategory(ctx context.Context, categoryID string) (*model.Category, error) {
	return s.categoryRepo.FindByID(ctx, categoryID)
}

func (s *categoryService) GetAllCategories(ctx context.Context) ([]*model.Category, error) {
	return s.categoryRepo.FindAll(ctx)
}

func (s *categoryService) UpdateCategory(ctx context.Context, categoryID, name, description string) (*model.Category, error) {
	category, err := s.categoryRepo.FindByID(ctx, categoryID)
	if err != nil {
		return nil, err
	}

	if name != "" {
		category.Name = name
	}
	if description != "" {
		category.Description = description
	}
	category.UpdatedAt = time.Now()

	if err := s.categoryRepo.Update(ctx, category); err != nil {
		s.logger.Error("Failed to update category:", err)
		return nil, err
	}
	s.logger.Info("Updated category:", category.ID)
	return category, nil
}

func (s *categoryService) DeleteCategory(ctx context.Context, categoryID string) error {
	category, err := s.categoryRepo.FindByID(ctx, categoryID)
	if err != nil {
		return err
	}

	if err := s.categoryRepo.Delete(ctx, category.ID); err != nil {
		s.logger.Error("Failed to delete category:", err)
		return err
	}
	s.logger.Info("Deleted category:", category.ID)
	return nil
}
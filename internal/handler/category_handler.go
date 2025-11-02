package handler

import (
	"net/http"

	"jump-challenge/internal/service"

	"github.com/labstack/echo/v4"
)

type CategoryHandler struct {
	categoryService service.CategoryService
	authHandler     *AuthHandler
	logger          echo.Logger
}

func NewCategoryHandler(categoryService service.CategoryService, authHandler *AuthHandler, logger echo.Logger) *CategoryHandler {
	return &CategoryHandler{
		categoryService: categoryService,
		authHandler:     authHandler,
		logger:          logger,
	}
}

// CreateCategory creates a new category
func (h *CategoryHandler) CreateCategory(c echo.Context) error {
	// Get the authenticated user
	user, err := h.authHandler.GetCurrentUser(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	// Parse the request body
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Validate input
	if req.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Name is required",
		})
	}

	// Create the category
	category, err := h.categoryService.CreateCategory(c.Request().Context(), user.ID, req.Name, req.Description)
	if err != nil {
		h.logger.Error("Failed to create category:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create category",
		})
	}

	return c.JSON(http.StatusCreated, category)
}

// GetCategory retrieves a category by ID
func (h *CategoryHandler) GetCategory(c echo.Context) error {
	categoryID := c.Param("id")

	category, err := h.categoryService.GetCategory(c.Request().Context(), categoryID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Category not found",
		})
	}

	// Return the category (shared among all users)
	return c.JSON(http.StatusOK, category)
}

// GetCategories retrieves all categories for the authenticated user
func (h *CategoryHandler) GetCategories(c echo.Context) error {
	// Get all categories (shared among all users)
	categories, err := h.categoryService.GetAllCategories(c.Request().Context())
	if err != nil {
		h.logger.Error("Failed to get categories:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get categories",
		})
	}

	return c.JSON(http.StatusOK, categories)
}

// UpdateCategory updates an existing category
func (h *CategoryHandler) UpdateCategory(c echo.Context) error {
	categoryID := c.Param("id")

	// Parse the request body
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Get the current category to check ownership
	_, err := h.categoryService.GetCategory(c.Request().Context(), categoryID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Category not found",
		})
	}

	// Update the category
	updatedCategory, err := h.categoryService.UpdateCategory(
		c.Request().Context(),
		categoryID,
		req.Name,
		req.Description,
	)
	if err != nil {
		h.logger.Error("Failed to update category:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update category",
		})
	}

	return c.JSON(http.StatusOK, updatedCategory)
}

// DeleteCategory deletes a category
func (h *CategoryHandler) DeleteCategory(c echo.Context) error {
	categoryID := c.Param("id")

	// Delete the category
	err := h.categoryService.DeleteCategory(c.Request().Context(), categoryID)
	if err != nil {
		h.logger.Error("Failed to delete category:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to delete category",
		})
	}

	return c.NoContent(http.StatusNoContent)
}

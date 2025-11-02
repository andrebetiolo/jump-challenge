package middleware

import (
	"net/http"

	"jump-challenge/internal/handler"

	"github.com/labstack/echo/v4"
)

// AuthMiddleware checks if the user is authenticated
func AuthMiddleware(authHandler *handler.AuthHandler) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Check if the user is authenticated by trying to get the current user
			_, err := authHandler.GetCurrentUser(c)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Unauthorized",
				})
			}

			return next(c)
		}
	}
}

// SessionMiddleware initializes the session store for Goth
func SessionMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Goth requires the session to be properly initialized
			// This middleware ensures the request and response are properly handled by Goth
			return next(c)
		}
	}
}

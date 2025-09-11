package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/xpanvictor/xarvis/internal/domains/user"
	"github.com/xpanvictor/xarvis/pkg/Logger"
)

// UserHandler handles user-related HTTP requests
type UserHandler struct {
	userService user.UserService
	logger      *Logger.Logger
}

// NewUserHandler creates a new user handler
func NewUserHandler(userService user.UserService, logger *Logger.Logger) *UserHandler {
	return &UserHandler{
		userService: userService,
		logger:      logger,
	}
}

// Register handles user registration
// @Summary Register a new user
// @Description Register a new user account with email and password
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body user.CreateUserRequest true "User registration data"
// @Success 201 {object} RegisterResponse "User registered successfully"
// @Failure 400 {object} ErrorResponse "Invalid request data"
// @Failure 409 {object} ErrorResponse "Email already exists"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /auth/register [post]
func (h *UserHandler) Register(c *gin.Context) {
	var req user.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request data",
			Details: err.Error(),
		})
		return
	}

	userResponse, err := h.userService.Register(c.Request.Context(), req)
	if err != nil {
		switch err {
		case user.ErrEmailAlreadyExists:
			c.JSON(http.StatusConflict, ErrorResponse{Error: "Email already exists"})
		default:
			h.logger.Errorf("registration error: %v", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		}
		return
	}

	c.JSON(http.StatusCreated, RegisterResponse{
		Message: "User registered successfully",
		User:    *userResponse,
	})
}

// Login handles user login
// @Summary User login
// @Description Authenticate user with email and password
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body user.LoginRequest true "User login credentials"
// @Success 200 {object} LoginResponse "Login successful with user data and tokens"
// @Failure 400 {object} ErrorResponse "Invalid request data"
// @Failure 401 {object} ErrorResponse "Invalid credentials"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /auth/login [post]
func (h *UserHandler) Login(c *gin.Context) {
	var req user.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request data",
			Details: err.Error(),
		})
		return
	}

	userResponse, tokens, err := h.userService.Login(c.Request.Context(), req)
	if err != nil {
		switch err {
		case user.ErrInvalidCredentials:
			c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Invalid credentials"})
		default:
			h.logger.Errorf("login error: %v", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		}
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		Message: "Login successful",
		User:    *userResponse,
		Tokens:  *tokens,
	})
}

// RefreshToken handles token refresh
// @Summary Refresh access token
// @Description Refresh an expired access token using refresh token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body RefreshTokenRequest true "Refresh token data"
// @Success 200 {object} RefreshTokenResponse "Token refreshed successfully"
// @Failure 400 {object} ErrorResponse "Invalid request data"
// @Failure 401 {object} ErrorResponse "Invalid refresh token"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /auth/refresh [post]
func (h *UserHandler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request data",
			Details: err.Error(),
		})
		return
	}

	tokens, err := h.userService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		switch err {
		case user.ErrInvalidToken:
			c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Invalid refresh token"})
		default:
			h.logger.Errorf("token refresh error: %v", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		}
		return
	}

	c.JSON(http.StatusOK, RefreshTokenResponse{
		Message: "Token refreshed successfully",
		Tokens:  *tokens,
	})
}

// GetProfile handles getting user profile
// @Summary Get user profile
// @Description Get the current authenticated user's profile
// @Tags User Profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} ProfileResponse "User profile data"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 404 {object} ErrorResponse "User not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /user/profile [get]
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID := c.GetString("userID") // From JWT middleware
	if userID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "User not authenticated"})
		return
	}

	userResponse, err := h.userService.GetProfile(c.Request.Context(), userID)
	if err != nil {
		switch err {
		case user.ErrUserNotFound:
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "User not found"})
		default:
			h.logger.Errorf("get profile error: %v", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		}
		return
	}

	c.JSON(http.StatusOK, ProfileResponse{
		User: *userResponse,
	})
}

// UpdateProfile handles updating user profile
// @Summary Update user profile
// @Description Update the current authenticated user's profile
// @Tags User Profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body user.UpdateUserRequest true "Profile update data"
// @Success 200 {object} UpdateProfileResponse "Profile updated successfully"
// @Failure 400 {object} ErrorResponse "Invalid request data"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 404 {object} ErrorResponse "User not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /user/profile [put]
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID := c.GetString("userID") // From JWT middleware
	if userID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "User not authenticated"})
		return
	}

	var req user.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request data",
			Details: err.Error(),
		})
		return
	}

	userResponse, err := h.userService.UpdateProfile(c.Request.Context(), userID, req)
	if err != nil {
		switch err {
		case user.ErrUserNotFound:
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "User not found"})
		default:
			h.logger.Errorf("update profile error: %v", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		}
		return
	}

	c.JSON(http.StatusOK, UpdateProfileResponse{
		Message: "Profile updated successfully",
		User:    *userResponse,
	})
}

// DeleteAccount handles account deletion
// @Summary Delete user account
// @Description Delete the current authenticated user's account
// @Tags User Profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} DeleteAccountResponse "Account deleted successfully"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 404 {object} ErrorResponse "User not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /user/account [delete]
func (h *UserHandler) DeleteAccount(c *gin.Context) {
	userID := c.GetString("userID") // From JWT middleware
	if userID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "User not authenticated"})
		return
	}

	err := h.userService.DeleteAccount(c.Request.Context(), userID)
	if err != nil {
		switch err {
		case user.ErrUserNotFound:
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "User not found"})
		default:
			h.logger.Errorf("delete account error: %v", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		}
		return
	}

	c.JSON(http.StatusOK, DeleteAccountResponse{
		Message: "Account deleted successfully",
	})
}

// ListUsers handles listing users (admin endpoint)
// @Summary List users (Admin)
// @Description List all users with pagination (admin only)
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param offset query int false "Number of users to skip" default(0)
// @Param limit query int false "Number of users to return" default(20)
// @Success 200 {object} ListUsersResponse "List of users with pagination"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 403 {object} ErrorResponse "Admin access required"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /admin/users [get]
func (h *UserHandler) ListUsers(c *gin.Context) {
	// Parse query parameters
	offsetStr := c.DefaultQuery("offset", "0")
	limitStr := c.DefaultQuery("limit", "20")

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 20 // Default limit with max of 100
	}

	users, total, err := h.userService.ListUsers(c.Request.Context(), offset, limit)
	if err != nil {
		h.logger.Errorf("list users error: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, ListUsersResponse{
		Users: users,
		Pagination: PaginationInfo{
			Total:  total,
			Offset: offset,
			Limit:  limit,
		},
	})
}

// GetUserByID handles getting a user by ID (admin endpoint)
// @Summary Get user by ID (Admin)
// @Description Get a specific user by ID (admin only)
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Success 200 {object} UserByIDResponse "User data"
// @Failure 400 {object} ErrorResponse "User ID is required"
// @Failure 401 {object} ErrorResponse "User not authenticated"
// @Failure 403 {object} ErrorResponse "Admin access required"
// @Failure 404 {object} ErrorResponse "User not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /admin/users/{id} [get]
func (h *UserHandler) GetUserByID(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "User ID is required"})
		return
	}

	userResponse, err := h.userService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		switch err {
		case user.ErrUserNotFound:
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "User not found"})
		default:
			h.logger.Errorf("get user by ID error: %v", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		}
		return
	}

	c.JSON(http.StatusOK, UserByIDResponse{
		User: *userResponse,
	})
}

// RegisterUserRoutes registers all user-related routes
func (h *UserHandler) RegisterUserRoutes(r *gin.RouterGroup) {
	// Public routes (no authentication required)
	public := r.Group("/auth")
	{
		public.POST("/register", h.Register)
		public.POST("/login", h.Login)
		public.POST("/refresh", h.RefreshToken)
	}

	// Protected routes (authentication required)
	protected := r.Group("/user")
	protected.Use(AuthMiddleware(h.userService, h.logger))
	{
		protected.GET("/profile", h.GetProfile)
		protected.PUT("/profile", h.UpdateProfile)
		protected.DELETE("/account", h.DeleteAccount)
	}

	// Admin routes (admin role required)
	admin := r.Group("/admin/users")
	admin.Use(AuthMiddleware(h.userService, h.logger), AdminMiddleware())
	{
		admin.GET("", h.ListUsers)
		admin.GET("/:id", h.GetUserByID)
	}
}

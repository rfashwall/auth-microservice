package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pquerna/otp/totp"
)

// User represents a user in the system
type User struct {
	Username     string
	Password     string
	Secret       string
	TwoFAEnabled bool
}

var users = make(map[string]User)

func main() {

	router := gin.Default()
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	auth := router.Group("/auth")
	auth.POST("/signup", signUp)
	auth.POST("/login", login)
	auth.POST("/enable-2fa", enable2FA)
	auth.POST("/verify", verify2FA)

	router.Run(":8080")
}

// signUp handles user registration
func signUp(c *gin.Context) {
	var newUser User
	if err := c.ShouldBindJSON(&newUser); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if the user already exists
	if _, exists := users[newUser.Username]; exists {
		c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
		return
	}

	// Save the user in the map
	users[newUser.Username] = newUser
	c.JSON(http.StatusCreated, gin.H{"message": "User created successfully"})
}

// login handles user login and 2FA verification
func login(c *gin.Context) {
	var credentials struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&credentials); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if the user exists
	user, exists := users[credentials.Username]
	if !exists || user.Password != credentials.Password {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Login successful"})
}

// verify2FA handles 2FA code verification
func verify2FA(c *gin.Context) {
	var verification struct {
		Username string `json:"username"`
		Code     string `json:"code"`
	}

	if err := c.ShouldBindJSON(&verification); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if the user exists
	user, exists := users[verification.Username]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Verify the provided code
	valid := totp.Validate(verification.Code, user.Secret)
	if !valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid 2FA code"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "2FA code is valid"})
}

// enable2FA generates a secret and QR code for 2FA setup
func enable2FA(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if the user exists
	user, exists := users[req.Username]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Generate a secret for the user
	secret, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "MyApp",
		AccountName: user.Username,
		Period:      60,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error generating QR code"})
		return
	}

	// Save the secret in the user object
	user.Secret = secret.Secret()
	user.TwoFAEnabled = true
	users[req.Username] = user

	c.JSON(http.StatusOK, gin.H{"secret": secret.Secret(), "url": secret.URL()})
}

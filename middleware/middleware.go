package mw

import (
	"book_talk/internal/models" // Import the Response model
	"context"
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"os"
	"time"
)

var secretKey = []byte(os.Getenv("SECRET_KEY"))

type Claims struct {
	Email     string `json:"email"`
	TokenType string `json:"tokenType"` // Это поле будет указывать на тип токена
	jwt.RegisteredClaims
}

// Protect is a middleware that ensures the user is authenticated by checking the access token
func Protect(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract the token from the Authorization header
		accessToken, err := ExtractAccessToken(r)
		if err != nil {
			// If the token is missing or invalid, send an error response with Response
			response := models.Response{
				Success:           false,
				Message:           "Неверный или отсутствующий токен",
				ErrorsDescription: []string{"Токен не передан или он неверен"},
			}
			SendJSONResponse(w, &response, http.StatusUnauthorized)
			return
		}

		// Validate the token
		email, err := ValidateAccessToken(accessToken)
		if err != nil {
			// If the token is invalid, send an error response with Response
			response := models.Response{
				Success:           false,
				Message:           "Невалидный токен",
				ErrorsDescription: []string{"Токен истек или имеет неверный формат"},
			}
			SendJSONResponse(w, &response, http.StatusUnauthorized)
			return
		}

		// Optionally, pass the email in the request context
		ctx := context.WithValue(r.Context(), "email", email)
		r = r.WithContext(ctx)

		// Call the next handler
		next(w, r)
	}
}

func GenerateTokens(email string) (string, string, error) {
	accessExpiration := time.Now().Add(2 * time.Hour)   // Access токен живет 1 час
	refreshExpiration := time.Now().Add(24 * time.Hour) // Refresh токен живет 24 часа

	// Генерация access токена
	accessToken, err := generateToken(email, accessExpiration, "access")
	if err != nil {
		return "", "", err
	}

	// Генерация refresh токена
	refreshToken, err := generateToken(email, refreshExpiration, "refresh")
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func generateToken(email string, expirationTime time.Time, tokenType string) (string, error) {
	claims := &Claims{
		Email:     email,
		TokenType: tokenType, // Устанавливаем тип токена
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			Issuer:    "book_talk",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		return "", fmt.Errorf("ошибка при подписании токена: %v", err)
	}

	return tokenString, nil
}

// ExtractAccessToken извлекает и проверяет accessToken из заголовка Authorization
func ExtractAccessToken(r *http.Request) (string, error) {
	// Получаем токен из заголовка Authorization
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("authorization header not found")
	}

	// Понимаем, что формат должен быть "Bearer <access_token>"
	if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		return "", fmt.Errorf("invalid authorization header format")
	}

	// Извлекаем сам токен
	accessToken := authHeader[7:]
	return accessToken, nil
}

// ValidateAccessToken проверяет валидность access токена
func ValidateAccessToken(tokenString string) (string, error) {
	email, err := ValidateToken(tokenString, "access") // Проверяем, что это именно accessToken
	if err != nil {
		return "", fmt.Errorf("невалидный access токен: %v", err)
	}
	return email, nil
}

// ValidateToken проверяет валидность токена и возвращает claims
func ValidateToken(tokenString string, tokenType string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return secretKey, nil
	})

	if err != nil {
		return "", fmt.Errorf("невалидный токен: %v", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return "", fmt.Errorf("невалидные данные в токене")
	}

	// Проверяем тип токена
	if claims.TokenType != tokenType {
		return "", fmt.Errorf("неправильный тип токена")
	}

	return claims.Email, nil
}

// SendJSONResponse sends a JSON response with the given response data and status code
func SendJSONResponse(w http.ResponseWriter, response *models.Response, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, `{"success": false, "message": "Failed to encode response"}`, http.StatusInternalServerError)
	}
}

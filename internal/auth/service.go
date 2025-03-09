package auth

import (
	"book_talk/internal/models"
	"database/sql"
	"errors"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var secretKey = []byte(os.Getenv("SECRET_KEY"))

type Claims struct {
	Email     string `json:"email"`
	TokenType string `json:"tokenType"` // Это поле будет указывать на тип токена
	jwt.RegisteredClaims
}

type Service struct {
	DB *sql.DB
}

func NewAuthService(db *sql.DB) *Service {
	return &Service{DB: db}
}

// Функция для валидации email
func isValidEmail(email string) bool {
	// Простой регулярное выражение для валидации email
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return re.MatchString(email)
}

// Функция для валидации имени и фамилии (только буквы)
func isValidName(name string) bool {
	re := regexp.MustCompile(`^[A-Za-zА-Яа-яЁё]+$`) // Только буквы (русские и латинские)
	return re.MatchString(name)
}

// Функция для валидации пароля
func isValidPassword(password string) bool {
	return len(password) >= 5
}

func (as *Service) RegisterUser(email, password, firstName, lastName string) (*models.Response, error) {
	// Валидация пустых строк для каждого поля
	if email == "" {
		return &models.Response{
			Success:           false,
			Message:           "Произошла ошибки при регистрации",
			ErrorsDescription: []string{"Email не может быть пустым"},
		}, nil
	}

	if password == "" {
		return &models.Response{
			Success:           false,
			Message:           "Произошла ошибки при регистрации",
			ErrorsDescription: []string{"Пароль не может быть пустым"},
		}, nil
	}

	if firstName == "" {
		return &models.Response{
			Success:           false,
			Message:           "Произошла ошибки при регистрации",
			ErrorsDescription: []string{"Имя не может быть пустым"},
		}, nil
	}

	if lastName == "" {
		return &models.Response{
			Success:           false,
			Message:           "Произошла ошибки при регистрации",
			ErrorsDescription: []string{"Фамилия не может быть пустой"},
		}, nil
	}

	// Валидация email
	if !isValidEmail(email) {
		return &models.Response{
			Success:           false,
			Message:           "Произошла ошибки при регистрации",
			ErrorsDescription: []string{"Неверный формат email"},
		}, nil
	}

	// Валидация имени и фамилии
	if !isValidName(firstName) || !isValidName(lastName) {
		return &models.Response{
			Success:           false,
			Message:           "Произошла ошибки при регистрации",
			ErrorsDescription: []string{"Имя и фамилия могут содержать только буквы"},
		}, nil
	}

	// Валидация пароля
	if !isValidPassword(password) {
		return &models.Response{
			Success:           false,
			Message:           "Произошла ошибки при регистрации",
			ErrorsDescription: []string{"Пароль должен содержать минимум 5 символов"},
		}, nil
	}

	// Хешируем пароль
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return &models.Response{
			Success:           false,
			Message:           "Произошла ошибки при регистрации",
			ErrorsDescription: []string{fmt.Sprintf("ошибка хеширования пароля: %v", err)},
		}, nil
	}

	// Проверяем, существует ли уже пользователь с таким email
	var existingEmail string
	err = as.DB.QueryRow("SELECT email FROM users WHERE email = $1", email).Scan(&existingEmail)
	if err == nil {
		// Если err == nil, значит пользователь с таким email уже существует
		return &models.Response{
			Success:           false,
			Message:           "Произошла ошибки при регистрации",
			ErrorsDescription: []string{"Пользователь с таким email уже существует"},
		}, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		// Если произошла другая ошибка, то возвращаем её
		return &models.Response{
			Success:           false,
			Message:           "Произошла ошибки при регистрации",
			ErrorsDescription: []string{fmt.Sprintf("ошибка при проверке email: %v", err)},
		}, nil
	}

	// Сохраняем пользователя в базе данных
	_, err = as.DB.Exec("INSERT INTO users (email, password, first_name, last_name) VALUES ($1, $2, $3, $4)",
		email, hashedPassword, firstName, lastName)
	if err != nil {
		return &models.Response{
			Success:           false,
			Message:           "Произошла ошибки при регистрации",
			ErrorsDescription: []string{fmt.Sprintf("ошибка сохранения пользователя: %v", err)},
		}, nil
	}

	// Создаем объект пользователя
	user := models.ShortUserInfo{
		Email:     email,
		FirstName: firstName,
		LastName:  lastName,
	}

	// Возвращаем успешный ответ с данными о пользователе
	response := &models.Response{
		Success:           true,
		Message:           "Успешно зарегестрирован",
		Data:              map[string]models.ShortUserInfo{"user": user},
		ErrorsDescription: nil,
	}

	return response, nil
}

func (as *Service) LoginUser(email, password string) (*models.Response, error) {
	var hashedPassword string
	err := as.DB.QueryRow("SELECT password FROM users WHERE email = $1", email).Scan(&hashedPassword)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &models.Response{
				Success:           false,
				Message:           "Произошла ошибка авторизации",
				ErrorsDescription: []string{"пользователь не найден"},
			}, nil
		}
		return &models.Response{
			Success:           false,
			Message:           "Произошла ошибка авторизации",
			ErrorsDescription: []string{fmt.Sprintf("ошибка при поиске пользователя: %v", err)},
		}, nil
	}

	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		return &models.Response{
			Success:           false,
			Message:           "Произошла ошибка авторизации",
			ErrorsDescription: []string{"неверный пароль"},
		}, nil
	}

	accessToken, refreshToken, err := GenerateTokens(email)
	if err != nil {
		return &models.Response{
			Success:           false,
			Message:           "Произошла ошибка авторизации",
			ErrorsDescription: []string{fmt.Sprintf("ошибка при генерации токенов: %v", err)},
		}, nil
	}

	// Return the success response with tokens
	response := &models.Response{
		Success:           true,
		Message:           "Успешная авторизация",
		Data:              map[string]string{"accessToken": accessToken, "refreshToken": refreshToken},
		ErrorsDescription: nil,
	}

	return response, nil
}

func GenerateTokens(email string) (string, string, error) {
	accessExpiration := time.Now().Add(15 * time.Minute) // Access токен живет 15 минут
	refreshExpiration := time.Now().Add(24 * time.Hour)  // Refresh токен живет 24 часа

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
		return "", fmt.Errorf("Authorization header not found")
	}

	// Понимаем, что формат должен быть "Bearer <access_token>"
	if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		return "", fmt.Errorf("Invalid authorization header format")
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

func (as *Service) RefreshToken(refreshToken string) (*models.Response, error) {
	// Проверяем валидность refresh токена
	email, err := ValidateToken(refreshToken, "refresh") // Передаем "refresh" в качестве типа токена
	if err != nil {
		return &models.Response{
			Success:           false,
			Message:           "Произошла ошибка обновления токена",
			ErrorsDescription: []string{"Невалидный refresh токен"},
		}, nil
	}

	// Генерируем новый accessToken
	accessToken, _, err := GenerateTokens(email)
	if err != nil {
		return &models.Response{
			Success:           false,
			Message:           "Произошла ошибка обновления токена",
			ErrorsDescription: []string{fmt.Sprintf("ошибка при генерации токенов: %v", err)},
		}, nil
	}

	// Возвращаем новый accessToken
	response := &models.Response{
		Success:           true,
		Message:           "Токен обновлен",
		Data:              map[string]string{"accessToken": accessToken},
		ErrorsDescription: nil,
	}

	return response, nil
}

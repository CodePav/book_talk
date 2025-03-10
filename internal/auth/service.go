package auth

import (
	"book_talk/internal/models"
	mw "book_talk/middleware"
	"database/sql"
	"errors"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"regexp"
	"unicode"
)

type Service struct {
	DB *sql.DB
}

func NewAuthService(db *sql.DB) *Service {
	return &Service{DB: db}
}

var (
	ErrUserAlreadyExists = errors.New("пользователь с таким email уже существует")
	ErrInvalidEmail      = errors.New("неверный формат email")
	ErrInvalidPassword   = errors.New("пароль должен содержать минимум 5 символов")
	ErrInvalidName       = errors.New("имя и фамилия могут содержать только буквы")
)

func (as *Service) RegisterUser(email, password, firstName, lastName string) (*models.Response, error) {
	// Валидация входных данных
	if email == "" {
		return nil, errors.New("email не может быть пустым")
	}
	if password == "" {
		return nil, errors.New("пароль не может быть пустым")
	}
	if firstName == "" {
		return nil, errors.New("имя не может быть пустым")
	}
	if lastName == "" {
		return nil, errors.New("фамилия не может быть пустой")
	}

	if !isValidEmail(email) {
		return nil, ErrInvalidEmail
	}
	if !isValidName(firstName) || !isValidName(lastName) {
		return nil, ErrInvalidName
	}
	// Проверка пароля
	isValid, errPass := IsValidPassword(password)
	if !isValid {
		return nil, fmt.Errorf("invalid password: %v", errPass)
	}

	// Проверяем, существует ли пользователь
	var existingEmail string
	err := as.DB.QueryRow("SELECT email FROM users WHERE email = $1", email).Scan(&existingEmail)
	if err == nil {
		return nil, ErrUserAlreadyExists
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("ошибка при проверке email")
	}

	// Хешируем пароль
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("ошибка хеширования пароля")
	}

	// Сохраняем пользователя в базе данных
	_, err = as.DB.Exec("INSERT INTO users (email, password, first_name, last_name) VALUES ($1, $2, $3, $4)",
		email, hashedPassword, firstName, lastName)
	if err != nil {
		return nil, errors.New("ошибка сохранения пользователя")
	}

	// Создаем объект пользователя
	user := models.ShortUserResponse{
		Email:     email,
		FirstName: firstName,
		LastName:  lastName,
	}

	// Возвращаем успешный ответ
	response := &models.Response{
		Message: "Успешно зарегистрирован",
		Data:    map[string]models.ShortUserResponse{"user": user},
	}

	return response, nil
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

func IsValidPassword(password string) (bool, error) {
	if len(password) < 5 {
		return false, fmt.Errorf("password too short, minimum 5 characters required")
	}

	// Check for uppercase letter, lowercase letter, number, and special character
	var hasUpper, hasLower, hasNumber, hasSpecial bool
	for _, c := range password {
		switch {
		case unicode.IsUpper(c):
			hasUpper = true
		case unicode.IsLower(c):
			hasLower = true
		case unicode.IsDigit(c):
			hasNumber = true
		case !unicode.IsLetter(c) && !unicode.IsDigit(c):
			hasSpecial = true
		}
	}

	if !hasUpper || !hasLower || !hasNumber || !hasSpecial {
		return false, fmt.Errorf("password must contain an uppercase letter, lowercase letter, number, and special character")
	}

	return true, nil
}

func (as *Service) LoginUser(email, password string) (*models.Response, error) {
	var (
		hashedPassword        string
		credentialsNonExpired bool
		accountNonExpired     bool
		accountNonLocked      bool
		enabled               bool
	)

	// Получаем хеш пароля и статус пользователя из базы данных
	err := as.DB.QueryRow("SELECT password, credentials_non_expired, account_non_expired, account_non_locked, enabled FROM users WHERE email = $1", email).
		Scan(&hashedPassword, &credentialsNonExpired, &accountNonExpired, &accountNonLocked, &enabled)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("пользователь не найден")
		}
		return nil, fmt.Errorf("ошибка при поиске пользователя")
	}

	// Проверяем, активна ли учетная запись
	if !credentialsNonExpired {
		return nil, fmt.Errorf("учетные данные недействительны")
	}

	if !accountNonExpired {
		return nil, fmt.Errorf("аккаунт выведен недействителен")
	}

	if !accountNonLocked {
		return nil, fmt.Errorf("аккаунт заблокирован")
	}

	if !enabled {
		return nil, fmt.Errorf("аккаунт не активирован")
	}

	// Сравниваем пароли
	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		return nil, fmt.Errorf("неверный пароль")
	}

	// Генерация токенов
	accessToken, refreshToken, err := mw.GenerateTokens(email)
	if err != nil {
		return nil, fmt.Errorf("ошибка при генерации ключей авторизации")
	}

	// Возвращаем успешный ответ с токенами
	return &models.Response{
		Message: "Успешная авторизация",
		Data:    map[string]string{"accessToken": accessToken, "refreshToken": refreshToken},
	}, nil
}

func (ah *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	// Получаем refreshToken из заголовков
	refreshToken := r.Header.Get("Refresh-Token")

	if refreshToken == "" {
		// Если refreshToken отсутствует, отправляем ошибку с 400
		response := &models.Response{
			Message: "Refresh-Token не найден в заголовках",
		}
		mw.SendJSONResponse(w, response, http.StatusBadRequest)
		return
	}

	// Пытаемся обновить токен
	response, err := ah.AuthService.Refresh(refreshToken)
	if err != nil {
		// Если ошибка, отправляем ошибочный ответ с 401
		response = &models.Response{
			Message: err.Error(),
		}
		mw.SendJSONResponse(w, response, http.StatusUnauthorized)
		return
	}

	// Если все прошло успешно, отправляем новый accessToken с 200
	mw.SendJSONResponse(w, response, http.StatusOK)
}

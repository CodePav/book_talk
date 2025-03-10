package auth

import (
	"book_talk/internal/models"
	mw "book_talk/middleware"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

type Handler struct {
	AuthService *Service
}

func NewAuthHandler(db *sql.DB) *Handler {
	return &Handler{
		AuthService: NewAuthService(db),
	}
}

func (ah *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email     string `json:"email"`
		Password  string `json:"password"`
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
	}

	// Декодируем JSON
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		mw.SendJSONResponse(w, &models.Response{Message: "Некорректный JSON"}, http.StatusBadRequest)
		return
	}

	// Вызываем сервис
	response, err := ah.AuthService.RegisterUser(req.Email, req.Password, req.FirstName, req.LastName)

	// Обрабатываем ошибки и отправляем правильный статус-код
	if err != nil {
		switch {
		case errors.Is(err, ErrUserAlreadyExists):
			mw.SendJSONResponse(w, &models.Response{Message: err.Error()}, http.StatusConflict) // 409
		case errors.Is(err, ErrInvalidEmail), errors.Is(err, ErrInvalidPassword), errors.Is(err, ErrInvalidName):
			mw.SendJSONResponse(w, &models.Response{Message: err.Error()}, http.StatusBadRequest) // 400
		default:
			mw.SendJSONResponse(w, &models.Response{Message: "Внутренняя ошибка сервера:" + err.Error()}, http.StatusInternalServerError) // 500
		}
		return
	}

	// Успешная регистрация
	mw.SendJSONResponse(w, response, http.StatusCreated)
}

func (ah *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Отправляем ошибку некорректного JSON
		response := &models.Response{
			Message: "Некорректный JSON",
		}
		mw.SendJSONResponse(w, response, http.StatusBadRequest)
		return
	}

	// Пытаемся авторизовать пользователя
	response, err := ah.AuthService.LoginUser(req.Email, req.Password)
	if err != nil {
		// Если произошла ошибка, отправляем ошибочный ответ с сообщением
		response = &models.Response{
			Message: err.Error(),
		}
		mw.SendJSONResponse(w, response, http.StatusUnauthorized)
		return
	}

	// Если все прошло успешно, отправляем успешный ответ
	mw.SendJSONResponse(w, response, http.StatusOK)
}

func (as *Service) Refresh(refreshToken string) (*models.Response, error) {
	// Проверяем валидность refresh токена
	email, err := mw.ValidateToken(refreshToken, "refresh") // Передаем "refresh" в качестве типа токена
	if err != nil {
		return nil, fmt.Errorf("передан невалидный refresh токен")
	}

	// Генерируем новый accessToken
	accessToken, _, err := mw.GenerateTokens(email)
	if err != nil {
		return nil, fmt.Errorf("ошибка обновления токена")
	}

	// Возвращаем успешный ответ с новым токеном
	return &models.Response{
		Message: "Токен обновлен",
		Data:    map[string]string{"accessToken": accessToken},
	}, nil
}

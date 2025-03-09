package auth

import (
	"database/sql"
	"encoding/json"
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

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Некорректный JSON", http.StatusBadRequest)
		return
	}

	response, err := ah.AuthService.RegisterUser(req.Email, req.Password, req.FirstName, req.LastName)
	if err != nil {
		// Here we return the error message from the response
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Return the response with user data
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (ah *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Некорректный JSON", http.StatusBadRequest)
		return
	}

	response, err := ah.AuthService.LoginUser(req.Email, req.Password)
	if err != nil {
		// Here we return the error message from the response
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Return the response with access tokens
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (ah *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	// Извлекаем refreshToken из заголовков
	refreshToken := r.Header.Get("Refresh-Token")

	if refreshToken == "" {
		http.Error(w, "Refresh-Token не найден в заголовках", http.StatusBadRequest)
		return
	}

	// Проверяем и обновляем токен
	response, err := ah.AuthService.RefreshToken(refreshToken)
	if err != nil {
		// Если ошибка при валидации токена, отправляем 401
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Если всё хорошо, возвращаем новый accessToken
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

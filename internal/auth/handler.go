package auth

import (
	"book_talk/internal/models"
	mw "book_talk/middleware"
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
		response := &models.Response{
			Success: false,
			Message: "Некорректный JSON",
		}
		mw.SendJSONResponse(w, response, http.StatusBadRequest)
		return
	}

	response, err := ah.AuthService.RegisterUser(req.Email, req.Password, req.FirstName, req.LastName)
	if err != nil {
		mw.SendJSONResponse(w, response, http.StatusInternalServerError)
		return
	}

	mw.SendJSONResponse(w, response, http.StatusCreated)
}

func (ah *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response := &models.Response{
			Success: false,
			Message: "Некорректный JSON",
		}
		mw.SendJSONResponse(w, response, http.StatusBadRequest)
		return
	}

	response, err := ah.AuthService.LoginUser(req.Email, req.Password)
	if err != nil {
		mw.SendJSONResponse(w, response, http.StatusUnauthorized)
		return
	}

	mw.SendJSONResponse(w, response, http.StatusOK)
}

func (ah *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	refreshToken := r.Header.Get("Refresh-Token")

	if refreshToken == "" {
		response := &models.Response{
			Success: false,
			Message: "Refresh-Token не найден в заголовках",
		}
		mw.SendJSONResponse(w, response, http.StatusBadRequest)
		return
	}

	response, err := ah.AuthService.RefreshToken(refreshToken)
	if err != nil {
		mw.SendJSONResponse(w, response, http.StatusUnauthorized)
		return
	}

	mw.SendJSONResponse(w, response, http.StatusOK)
}

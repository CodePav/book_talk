package users

import (
	"book_talk/internal/models"
	mw "book_talk/middleware"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Handler struct {
	UserService *Service
}

// Новый хэндлер для инициализации с сервисом
func NewUsersHandler(db *sql.DB) *Handler {
	return &Handler{
		UserService: NewUsersService(db),
	}
}

// Получение всех пользователей
func (h *Handler) GetAllUsers(w http.ResponseWriter, r *http.Request) {
	response, err := h.UserService.GetAllUsers()
	if err != nil {
		response = &models.Response{
			Success:           false,
			Message:           "Error fetching users",
			ErrorsDescription: fmt.Sprintf("Service error: %v", err),
		}
		mw.SendJSONResponse(w, response, http.StatusInternalServerError)
		return
	}
	mw.SendJSONResponse(w, response, http.StatusOK)
}

// Получение текущего пользователя
func (h *Handler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	email, ok := r.Context().Value("email").(string)
	if !ok {
		mw.SendJSONResponse(w, &models.Response{
			Success: false, Message: "Unauthorized", ErrorsDescription: "Invalid email in context",
		}, http.StatusUnauthorized)
		return
	}

	response, err := h.UserService.GetUser(email)
	if err != nil {
		response = &models.Response{
			Success:           false,
			Message:           "Error fetching user data",
			ErrorsDescription: fmt.Sprintf("Service error: %v", err),
		}
		mw.SendJSONResponse(w, response, http.StatusInternalServerError)
		return
	}
	mw.SendJSONResponse(w, response, http.StatusOK)
}

// Обновление пользователя
func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	var updatedUser models.UserDTO
	if err := json.NewDecoder(r.Body).Decode(&updatedUser); err != nil {
		mw.SendJSONResponse(w, &models.Response{
			Success: false, Message: "Invalid request body", ErrorsDescription: err.Error(),
		}, http.StatusBadRequest)
		return
	}

	response, err := h.UserService.UpdateUser(updatedUser)
	if err != nil {
		mw.SendJSONResponse(w, response, http.StatusInternalServerError)
		return
	}
	mw.SendJSONResponse(w, response, http.StatusOK)
}

// Получение изображения пользователя
func (h *Handler) GetUserImage(w http.ResponseWriter, r *http.Request) {
	email, ok := r.Context().Value("email").(string)
	if !ok {
		mw.SendJSONResponse(w, &models.Response{
			Success: false, Message: "Unauthorized", ErrorsDescription: "Invalid email in context",
		}, http.StatusUnauthorized)
		return
	}

	response, err := h.UserService.GetUserImage(email)
	if err != nil {
		mw.SendJSONResponse(w, response, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/jpeg")
	w.WriteHeader(http.StatusOK)
	w.Write(response.Data.([]byte))
}

// Обновление изображения пользователя
func (h *Handler) UpdateUserImage(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	email, ok := r.Context().Value("email").(string)
	if !ok {
		mw.SendJSONResponse(w, &models.Response{
			Success: false, Message: "Unauthorized", ErrorsDescription: "Invalid email in context",
		}, http.StatusUnauthorized)
		return
	}

	imageData, err := io.ReadAll(r.Body)
	if err != nil {
		mw.SendJSONResponse(w, &models.Response{
			Success: false, Message: "Failed to read image data", ErrorsDescription: err.Error(),
		}, http.StatusBadRequest)
		return
	}

	response, err := h.UserService.UpdateUserImage(imageData, email)
	if err != nil {
		mw.SendJSONResponse(w, response, http.StatusInternalServerError)
		return
	}
	mw.SendJSONResponse(w, response, http.StatusOK)
}

// Смена пароля
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	email, ok := r.Context().Value("email").(string)
	if !ok {
		mw.SendJSONResponse(w, &models.Response{
			Success: false, Message: "Unauthorized", ErrorsDescription: "Invalid email in context",
		}, http.StatusUnauthorized)
		return
	}

	var passwordData struct {
		OldPassword string `json:"oldPassword"`
		NewPassword string `json:"newPassword"`
	}
	if err := json.NewDecoder(r.Body).Decode(&passwordData); err != nil {
		mw.SendJSONResponse(w, &models.Response{
			Success: false, Message: "Invalid request body", ErrorsDescription: err.Error(),
		}, http.StatusBadRequest)
		return
	}

	response, err := h.UserService.ChangePassword(passwordData.OldPassword, passwordData.NewPassword, email)
	if err != nil {
		mw.SendJSONResponse(w, response, http.StatusInternalServerError)
		return
	}
	mw.SendJSONResponse(w, response, http.StatusOK)
}

// Удаление пользователя
func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	email, ok := r.Context().Value("email").(string)
	if !ok {
		mw.SendJSONResponse(w, &models.Response{
			Success: false, Message: "Unauthorized", ErrorsDescription: "Invalid email in context",
		}, http.StatusUnauthorized)
		return
	}

	response, err := h.UserService.DeleteUser(email)
	if err != nil {
		mw.SendJSONResponse(w, response, http.StatusInternalServerError)
		return
	}
	mw.SendJSONResponse(w, response, http.StatusNoContent)
}

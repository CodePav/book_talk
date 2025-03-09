package users

import (
	"book_talk/internal/models"
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

// GetCurrentUser для получения текущего пользователя
func (h *Handler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	response, err := h.UserService.GetCurrentUser()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// UpdateUser для обновления пользователя
func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	var updatedUser models.User
	if err := json.NewDecoder(r.Body).Decode(&updatedUser); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	response, err := h.UserService.UpdateUser(updatedUser)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetUserImage для получения изображения пользователя
func (h *Handler) GetUserImage(w http.ResponseWriter, r *http.Request) {
	image, err := h.UserService.GetUserImage()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/jpeg")
	w.Write(image)
}

// UpdateUserImage для обновления изображения пользователя
func (h *Handler) UpdateUserImage(w http.ResponseWriter, r *http.Request) {
	// Закрываем тело запроса после его обработки
	defer r.Body.Close()

	// Чтение данных изображения из тела запроса
	imageData, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to read image data: %v", err), http.StatusBadRequest)
		return
	}

	// Вызываем сервис для обновления изображения
	err = h.UserService.UpdateUserImage(imageData)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to update user image: %v", err), http.StatusInternalServerError)
		return
	}

	// Отправляем успешный ответ
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Image updated successfully"))
}

// ChangePassword для смены пароля
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var passwordData struct {
		OldPassword string `json:"oldPassword"`
		NewPassword string `json:"newPassword"`
	}
	if err := json.NewDecoder(r.Body).Decode(&passwordData); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := h.UserService.ChangePassword(passwordData.OldPassword, passwordData.NewPassword)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// GetUserBookingsPaginationHandler для получения информации о пагинации
func (h *Handler) GetUserBookingsPaginationHandler(w http.ResponseWriter, r *http.Request) {
	page, size := parsePaginationParams(r)

	// Формируем JSON-ответ
	response := Pagination{
		Page: page,
		Size: size,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// DeleteUser для удаления пользователя
func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	err := h.UserService.DeleteUser()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

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

func (h *Handler) GetAllUsers(w http.ResponseWriter, r *http.Request) {
	// Получаем всех пользователей
	response, err := h.UserService.GetAllUsers()
	if err != nil {
		// Если произошла ошибка при получении данных
		response = &models.Response{
			Success:           false,
			Message:           "Error fetching users",
			Data:              nil,
			ErrorsDescription: err.Error(), // Описание ошибки
		}
		// Отправляем ошибочный ответ с кодом 500
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		// Если все прошло успешно, отправляем успешный ответ
		w.WriteHeader(http.StatusOK)
	}

	// Отправляем JSON-ответ
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		// Если произошла ошибка при кодировании ответа
		http.Error(w, "Error encoding response to JSON", http.StatusInternalServerError)
	}
}

// GetCurrentUser для получения текущего пользователя
func (h *Handler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	// Извлекаем email из контекста
	email := r.Context().Value("email").(string)

	response, err := h.UserService.GetUser(email)
	if err != nil {
		// Ошибка запроса или других операций
		response = &models.Response{
			Success:           false,
			Message:           "Error fetching user data",
			Data:              nil,
			ErrorsDescription: err.Error(),
		}
	}

	// Отправляем JSON-ответ
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// UpdateUser для обновления пользователя
func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	var updatedUser models.UserDTO

	// Декодируем запрос, переданный от клиента
	if err := json.NewDecoder(r.Body).Decode(&updatedUser); err != nil {
		// В случае ошибки в теле запроса, отправляем ошибку
		http.Error(w, "Invalid request body. Please check the structure of the data.", http.StatusBadRequest)
		return
	}

	// Обновляем пользователя через сервис
	response, err := h.UserService.UpdateUser(updatedUser)
	if err != nil {
		// Если ошибка, отправляем статус 500 и сообщение об ошибке
		http.Error(w, fmt.Sprintf("Error updating user: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	// Устанавливаем заголовок типа контента и отправляем успешный ответ
	w.Header().Set("Content-Type", "application/json")

	// Обрабатываем ошибку при отправке ответа
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, fmt.Sprintf("Error encoding response: %s", err.Error()), http.StatusInternalServerError)
		return
	}
}

// GetUserImage для получения изображения пользователя
func (h *Handler) GetUserImage(w http.ResponseWriter, r *http.Request) {
	// Извлекаем email из контекста
	email := r.Context().Value("email").(string)

	// Получаем изображение через сервис
	response, err := h.UserService.GetUserImage(email)
	if err != nil {
		// Отправляем ошибку, если что-то пошло не так
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Если изображение успешно получено
	w.Header().Set("Content-Type", "image/jpeg")
	w.WriteHeader(http.StatusOK)
	w.Write(response.Data.([]byte))
}

// UpdateUserImage для обновления изображения пользователя
func (h *Handler) UpdateUserImage(w http.ResponseWriter, r *http.Request) {
	// Закрываем тело запроса после его обработки
	defer r.Body.Close()

	// Извлекаем email из контекста
	email := r.Context().Value("email").(string)

	// Чтение данных изображения из тела запроса
	imageData, err := io.ReadAll(r.Body)
	if err != nil {
		// Возвращаем ошибку через Response, если не удается прочитать данные изображения
		response := &models.Response{
			Success:           false,
			Message:           "Failed to read image data",
			Data:              nil,
			ErrorsDescription: fmt.Sprintf("Error reading image data: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Вызываем сервис для обновления изображения
	response, err := h.UserService.UpdateUserImage(imageData, email)
	if err != nil {
		// Возвращаем ошибку через Response, если не удается обновить изображение
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Отправляем успешный ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// ChangePassword для смены пароля
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	// Извлекаем email из контекста
	email := r.Context().Value("email").(string)

	var passwordData struct {
		OldPassword string `json:"oldPassword"`
		NewPassword string `json:"newPassword"`
	}
	if err := json.NewDecoder(r.Body).Decode(&passwordData); err != nil {
		// Возвращаем ошибку через Response, если не удается декодировать тело запроса
		response := &models.Response{
			Success:           false,
			Message:           "Invalid request body",
			Data:              nil,
			ErrorsDescription: fmt.Sprintf("Error reading request body: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Вызываем сервис для смены пароля
	response, err := h.UserService.ChangePassword(passwordData.OldPassword, passwordData.NewPassword, email)
	if err != nil {
		// Возвращаем ошибку через Response, если не удается сменить пароль
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Отправляем успешный ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// DeleteUser для удаления пользователя
func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	// Извлекаем email из контекста
	email := r.Context().Value("email").(string)

	// Вызываем сервис для удаления пользователя
	response, err := h.UserService.DeleteUser(email)
	if err != nil {
		// Если ошибка при удалении, отправляем ответ с ошибкой
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Отправляем успешный ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)
}

package users

import (
	"book_talk/internal/models"
	mw "book_talk/middleware"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
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
	// Пытаемся получить всех пользователей
	response, err := h.UserService.GetAllUsers()
	if err != nil {
		// Если произошла ошибка, отправляем ошибочный ответ с 500
		response = &models.Response{
			Message: err.Error(), // Добавляем описание ошибки
		}
		mw.SendJSONResponse(w, response, http.StatusInternalServerError)
		return
	}

	// Если все прошло успешно, отправляем список пользователей с 200
	mw.SendJSONResponse(w, response, http.StatusOK)
}

func (h *Handler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	// Извлекаем email из контекста
	email, ok := r.Context().Value("email").(string)
	if !ok {
		// Если email не найден, отправляем ошибку 401
		mw.SendJSONResponse(w, &models.Response{
			Message: "Unauthorized",
		}, http.StatusUnauthorized)
		return
	}

	// Пытаемся получить данные о пользователе
	response, err := h.UserService.GetUser(email)
	if err != nil {
		// Если ошибка при получении данных, отправляем ошибку 500
		response = &models.Response{
			Message: err.Error(),
		}
		mw.SendJSONResponse(w, response, http.StatusInternalServerError)
		return
	}

	// Если данные о пользователе получены, отправляем успешный ответ с 200
	mw.SendJSONResponse(w, response, http.StatusOK)
}

func (h *Handler) GetUserBookings(w http.ResponseWriter, r *http.Request) {
	// Получение параметров пагинации
	pageStr := r.URL.Query().Get("page")
	sizeStr := r.URL.Query().Get("size")

	// Преобразуем параметры пагинации в числа
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 0 {
		page = 0 // Если ошибка или отрицательное значение, ставим 0
	}

	size, err := strconv.Atoi(sizeStr)
	if err != nil || size <= 0 {
		size = 10 // По умолчанию 10 записей на странице
	}

	// Получаем текущего пользователя
	email, ok := r.Context().Value("email").(string)
	if !ok {
		// Если email отсутствует в контексте, отправляем ошибку 401
		mw.SendJSONResponse(w, &models.Response{
			Message: "Unauthorized",
		}, http.StatusUnauthorized)
		return
	}

	// Получаем данные о пользователе и его бронированиях
	bookings, err := h.UserService.GetUserBookings(email, page, size)
	if err != nil {
		// Если произошла ошибка в сервисе, отправляем ошибку 500
		mw.SendJSONResponse(w, &models.Response{
			Message: err.Error(),
		}, http.StatusInternalServerError)
		return
	}

	// Создаем ответ с бронированиями и отправляем его с кодом 200
	response := &models.Response{
		Message: "User bookings fetched successfully",
		Data:    map[string][]models.Booking{"bookings": bookings},
	}
	mw.SendJSONResponse(w, response, http.StatusOK)
}

func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	var updatedUser models.UserDTO
	if err := json.NewDecoder(r.Body).Decode(&updatedUser); err != nil {
		mw.SendJSONResponse(w, &models.Response{
			Message: "Invalid request body",
		}, http.StatusBadRequest)
		return
	}

	// Обновляем пользователя
	user, err := h.UserService.UpdateUser(updatedUser)
	if err != nil {
		// Обработка ошибок с описанием ошибки
		mw.SendJSONResponse(w, &models.Response{
			Message: err.Error(),
		}, http.StatusInternalServerError)
		return
	}

	// Возвращаем успешный ответ
	response := &models.Response{
		Message: "User updated successfully",
		Data:    map[string]models.UserDTO{"user": *user},
	}

	mw.SendJSONResponse(w, response, http.StatusOK)
}

func (h *Handler) GetUserImage(w http.ResponseWriter, r *http.Request) {
	// Извлекаем email пользователя из контекста
	email, ok := r.Context().Value("email").(string)
	if !ok {
		mw.SendJSONResponse(w, &models.Response{
			Message: "Unauthorized",
		}, http.StatusUnauthorized)
		return
	}

	// Получаем изображение пользователя
	response, err := h.UserService.GetUserImage(email)
	if err != nil {
		// В случае ошибки, возвращаем ответ с ошибкой
		mw.SendJSONResponse(w, response, http.StatusInternalServerError)
		return
	}

	// Устанавливаем тип контента для изображения
	w.Header().Set("Content-Type", "image/jpeg")
	w.WriteHeader(http.StatusOK)

	// Отправляем изображение в теле ответа
	w.Write(response.Data.([]byte))
}

// Обновление изображения пользователя
func (h *Handler) UpdateUserImage(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	email, ok := r.Context().Value("email").(string)
	if !ok {
		mw.SendJSONResponse(w, &models.Response{
			Message: "Unauthorized",
		}, http.StatusUnauthorized)
		return
	}

	imageData, err := io.ReadAll(r.Body)
	if err != nil {
		mw.SendJSONResponse(w, &models.Response{
			Message: "Failed to read image data",
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

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	email, ok := r.Context().Value("email").(string)
	if !ok {
		mw.SendJSONResponse(w, &models.Response{
			Message: "Unauthorized",
		}, http.StatusUnauthorized)
		return
	}

	// Парсим тело запроса с паролями
	var passwordData struct {
		OldPassword string `json:"oldPassword"`
		NewPassword string `json:"newPassword"`
	}
	if err := json.NewDecoder(r.Body).Decode(&passwordData); err != nil {
		mw.SendJSONResponse(w, &models.Response{
			Message: "Invalid request body",
		}, http.StatusBadRequest)
		return
	}

	// Вызываем сервис для изменения пароля
	response, err := h.UserService.ChangePassword(passwordData.OldPassword, passwordData.NewPassword, email)
	if err != nil {
		mw.SendJSONResponse(w, response, http.StatusInternalServerError)
		return
	}

	// Отправляем успешный ответ
	mw.SendJSONResponse(w, response, http.StatusOK)
}

func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	email, ok := r.Context().Value("email").(string)
	if !ok {
		mw.SendJSONResponse(w, &models.Response{
			Message: "Unauthorized",
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

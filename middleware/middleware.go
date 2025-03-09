package mw

import (
	"book_talk/internal/auth"
	"book_talk/internal/models" // Импортируем модель Response
	"context"
	"encoding/json"
	"net/http"
)

func Protect(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Извлекаем токен из заголовка Authorization
		accessToken, err := auth.ExtractAccessToken(r)
		if err != nil {
			// Если токен отсутствует или неверный, отправляем ошибку с Response
			response := models.Response{
				Success:           false,
				Message:           "Неверный или отсутствующий токен",
				ErrorsDescription: []string{"Токен не передан или он неверен"},
			}
			w.WriteHeader(http.StatusUnauthorized) // 401 Unauthorized
			json.NewEncoder(w).Encode(response)
			return
		}

		// Проверяем валидность токена
		email, err := auth.ValidateAccessToken(accessToken)
		if err != nil {
			// Если токен невалиден, отправляем ошибку с Response
			response := models.Response{
				Success:           false,
				Message:           "Невалидный токен",
				ErrorsDescription: []string{"Токен истек или имеет неверный формат"},
			}
			w.WriteHeader(http.StatusUnauthorized) // 401 Unauthorized
			json.NewEncoder(w).Encode(response)
			return
		}

		// При необходимости, можем передать email пользователя в контекст запроса
		ctx := context.WithValue(r.Context(), "email", email)
		r = r.WithContext(ctx)

		// Выполняем основной хэндлер
		next(w, r)
	}
}

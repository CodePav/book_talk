package users

import (
	"book_talk/internal/models"
	"database/sql"
	"errors"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"log"
	"net/http"
	"os"
	"strconv"
)

type Service struct {
	DB *sql.DB
}

func NewUsersService(db *sql.DB) *Service {
	return &Service{DB: db}
}

// Получить текущего пользователя с департаментом, ролями и бронированиями
func (s *Service) GetCurrentUser() (*models.Response, error) {
	var user models.User
	query := `SELECT email, first_name, last_name, password, department_id, image, theme, credentials_non_expired, account_non_expired, account_non_locked, enabled 
              FROM users WHERE email = $1`
	err := s.DB.QueryRow(query, "currentUser@example.com").Scan(
		&user.Email, &user.FirstName, &user.LastName, &user.Password, &user.Department.ID,
		&user.Image, &user.Theme, &user.CredentialsNonExpired, &user.AccountNonExpired,
		&user.AccountNonLocked, &user.Enabled,
	)
	if err != nil {
		return nil, err
	}

	// Получаем департамент
	departmentQuery := `SELECT id, name, short_name, color FROM department WHERE id = $1`
	err = s.DB.QueryRow(departmentQuery, user.Department.ID).Scan(
		&user.Department.ID, &user.Department.Name, &user.Department.ShortName, &user.Department.Color,
	)
	if err != nil {
		return nil, err
	}

	// Получаем связанные бронирования
	bookingsQuery := `SELECT id, room_id, user_email, time FROM booking WHERE user_email = $1`
	rows, err := s.DB.Query(bookingsQuery, user.Email)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var booking models.Booking
		if err := rows.Scan(&booking.ID, &booking.Room.ID, &booking.User.Email, &booking.Time); err != nil {
			return nil, err
		}
		booking.User = user // Присваиваем пользователя для бронирования
		user.Bookings = append(user.Bookings, booking)
	}

	// Получаем связанные роли
	rolesQuery := `SELECT id, authority FROM role WHERE user_email = $1`
	rows, err = s.DB.Query(rolesQuery, user.Email)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var role models.Role
		if err := rows.Scan(&role.ID, &role.Authority); err != nil {
			return nil, err
		}
		role.User = &user // Присваиваем пользователя для роли
		user.Roles = append(user.Roles, role)
	}

	// Возвращаем успешный ответ с данными о пользователе
	response := &models.Response{
		Success:           true,
		Message:           "",
		Data:              map[string]models.User{"user": user},
		ErrorsDescription: nil,
	}

	return response, nil
}

// Обновить пользователя, департамент, бронирования и роли
func (s *Service) UpdateUser(updatedUser models.User) (*models.Response, error) {
	// Обновляем информацию о пользователе
	query := `
		UPDATE users
		SET first_name = $1, last_name = $2, password = $3, department_id = $4, theme = $5, image = $6
		WHERE email = $7
		RETURNING email, first_name, last_name, password, department_id, image, theme, credentials_non_expired, account_non_expired, account_non_locked, enabled
	`
	var user models.User
	err := s.DB.QueryRow(query, updatedUser.FirstName, updatedUser.LastName, updatedUser.Password,
		updatedUser.Department.ID, updatedUser.Theme, updatedUser.Image, updatedUser.Email).Scan(
		&user.Email, &user.FirstName, &user.LastName, &user.Password, &user.Department.ID,
		&user.Image, &user.Theme, &user.CredentialsNonExpired, &user.AccountNonExpired,
		&user.AccountNonLocked, &user.Enabled,
	)
	if err != nil {
		return nil, err
	}

	// Обновляем департамент
	_, err = s.DB.Exec(`UPDATE department SET name = $1, short_name = $2, color = $3 WHERE id = $4`,
		updatedUser.Department.Name, updatedUser.Department.ShortName, updatedUser.Department.Color, updatedUser.Department.ID)
	if err != nil {
		return nil, err
	}

	// Обновляем бронирования
	_, err = s.DB.Exec(`DELETE FROM user_booking WHERE user_email = $1`, updatedUser.Email)
	if err != nil {
		return nil, err
	}

	for _, booking := range updatedUser.Bookings {
		_, err = s.DB.Exec(`INSERT INTO user_booking (user_email, booking_id) VALUES ($1, $2)`, updatedUser.Email, booking.ID)
		if err != nil {
			return nil, err
		}
	}

	// Обновляем роли
	_, err = s.DB.Exec(`DELETE FROM user_role WHERE user_email = $1`, updatedUser.Email)
	if err != nil {
		return nil, err
	}

	for _, role := range updatedUser.Roles {
		_, err = s.DB.Exec(`INSERT INTO user_role (user_email, role_id) VALUES ($1, $2)`, updatedUser.Email, role.ID)
		if err != nil {
			return nil, err
		}
	}

	// Возвращаем успешный ответ с данными о пользователе
	response := &models.Response{
		Success:           true,
		Message:           "",
		Data:              map[string]models.User{"user": user},
		ErrorsDescription: nil,
	}

	return response, nil
}

// Получить изображение пользователя
func (s *Service) GetUserImage() ([]byte, error) {
	// Получаем текущего пользователя
	userResponse, err := s.GetCurrentUser()
	if err != nil {
		return nil, err
	}

	// Приводим Data к типу map[string]User
	userData, ok := userResponse.Data.(map[string]interface{})["user"].(models.User)
	if !ok {
		return nil, errors.New("invalid data format")
	}

	// Проверка на наличие изображения
	imagePath := userData.Image
	if imagePath == "" {
		return nil, errors.New("image not found")
	}

	// Чтение изображения с файловой системы
	image, err := os.ReadFile(imagePath)
	if err != nil {
		return nil, err
	}

	return image, nil
}

func (s *Service) UpdateUserImage(imageData []byte) error {
	// Получаем текущего пользователя
	userResponse, err := s.GetCurrentUser()
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	// Приводим Data к типу map[string]interface{} и извлекаем User
	userData, ok := userResponse.Data.(map[string]interface{})["user"].(map[string]interface{})
	if !ok {
		return errors.New("invalid data format or missing 'user' field")
	}

	// Извлекаем информацию о пользователе, например, email
	email, ok := userData["email"].(string)
	if !ok {
		return errors.New("user email is missing or invalid")
	}

	// Формируем путь для изображения
	imagePath := fmt.Sprintf("images/%s.jpg", email)

	// Записываем изображение на диск
	err = os.WriteFile(imagePath, imageData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write image to disk: %w", err)
	}

	// Обновляем путь к изображению в базе данных
	query := `UPDATE users SET image = $1 WHERE email = $2`
	_, err = s.DB.Exec(query, imagePath, email)
	if err != nil {
		return fmt.Errorf("failed to update image path in database: %w", err)
	}

	return nil
}

func (s *Service) ChangePassword(oldPassword, newPassword string) error {
	// Получаем текущего пользователя
	userResponse, err := s.GetCurrentUser()
	if err != nil {
		return err
	}

	// Приводим Data к типу map[string]interface{} и извлекаем User
	userData, ok := userResponse.Data.(map[string]interface{})["user"].(models.User)
	if !ok {
		return errors.New("invalid data format")
	}

	// Проверка старого пароля
	err = bcrypt.CompareHashAndPassword([]byte(userData.Password), []byte(oldPassword))
	if err != nil {
		return errors.New("incorrect old password")
	}

	// Хэширование нового пароля
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Обновление пароля в базе данных
	query := `UPDATE users SET password = $1 WHERE email = $2`
	_, err = s.DB.Exec(query, string(hashedPassword), userData.Email)
	if err != nil {
		return err
	}

	return nil
}
func (s *Service) DeleteUser() error {
	// Получаем текущего пользователя
	userResponse, err := s.GetCurrentUser()
	if err != nil {
		return err
	}

	// Приводим Data к типу map[string]interface{} и извлекаем User
	userData, ok := userResponse.Data.(map[string]interface{})["user"].(models.User)
	if !ok {
		return errors.New("invalid data format")
	}

	// Удаляем связанные данные
	_, err = s.DB.Exec(`DELETE FROM user_booking WHERE user_email = $1`, userData.Email)
	if err != nil {
		return err
	}

	_, err = s.DB.Exec(`DELETE FROM user_role WHERE user_email = $1`, userData.Email)
	if err != nil {
		return err
	}

	// Удаляем пользователя из базы данных
	_, err = s.DB.Exec(`DELETE FROM users WHERE email = $1`, userData.Email)
	if err != nil {
		return err
	}

	// Удаляем изображение, если оно есть
	if userData.Image != "" {
		err = os.Remove(userData.Image)
		if err != nil {
			log.Println("Failed to delete user image:", err)
		}
	}

	return nil
}

// Структура Pagination для пагинации
type Pagination struct {
	Page int `json:"page"`
	Size int `json:"size"`
}

// parsePaginationParams извлекает параметры пагинации из запроса
func parsePaginationParams(r *http.Request) (int, int) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))

	if page < 0 {
		page = 0
	}
	if size < 1 {
		size = 10
	}
	return page, size
}

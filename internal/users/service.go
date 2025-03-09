package users

import (
	"book_talk/internal/models"
	"github.com/mitchellh/mapstructure"

	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"image"
	"log"
	"os"
)

type Service struct {
	DB *sql.DB
}

func NewUsersService(db *sql.DB) *Service {
	return &Service{DB: db}
}

func (s *Service) GetAllUsers() (*models.Response, error) {
	// Получаем список email'ов пользователей
	query := `SELECT email FROM users`
	rows, err := s.DB.Query(query)
	if err != nil {
		return &models.Response{
			Success:           false,
			Message:           "Error fetching user emails",
			ErrorsDescription: fmt.Sprintf("error fetching user emails: %v", err),
		}, err
	}
	defer rows.Close()

	var users []models.UserResponse
	var email string

	// Обходим все email'ы и вызываем GetUser
	for rows.Next() {
		if err := rows.Scan(&email); err != nil {
			return &models.Response{
				Success:           false,
				Message:           "Error scanning user email",
				ErrorsDescription: fmt.Sprintf("error scanning user email: %v", err),
			}, err
		}

		userResponse, err := s.GetUser(email)
		if err != nil {
			return &models.Response{
				Success:           false,
				Message:           "Error fetching user data",
				ErrorsDescription: fmt.Sprintf("error fetching user data for %s: %v", email, err),
			}, err
		}

		var userData models.UserResponse
		err = mapstructure.Decode(userResponse.Data, &userData)
		if err != nil {
			return &models.Response{
				Success:           false,
				Message:           "Invalid data format",
				ErrorsDescription: "Error converting user data",
			}, fmt.Errorf("failed to decode user data for email %s: %v", email, err)
		}
		// Если bookings и roles nil — заменяем на пустые списки
		if userData.Bookings == nil {
			userData.Bookings = []models.Booking{}
		}
		if userData.Roles == nil {
			userData.Roles = []models.Role{}
		}
		users = append(users, userData)
	}

	// Проверяем на ошибки при обходе данных
	if err := rows.Err(); err != nil {
		return &models.Response{
			Success:           false,
			Message:           "Error during rows iteration",
			ErrorsDescription: fmt.Sprintf("error during rows iteration: %v", err),
		}, err
	}

	// Возвращаем успешный ответ с пользователями
	return &models.Response{
		Success:           true,
		Message:           "Users fetched successfully",
		Data:              users,
		ErrorsDescription: nil,
	}, nil
}

func (s *Service) GetUser(email string) (*models.Response, error) {
	var userDTO models.UserDTO

	// 1. Получаем основную информацию о пользователе
	var departmentID sql.NullInt64 // Используем sql.NullInt64 для обработки NULL значений
	query := `SELECT email, first_name, last_name, password, department_id, image, theme, 
                     credentials_non_expired, account_non_expired, account_non_locked, enabled
              FROM users WHERE email = $1`
	err := s.DB.QueryRow(query, email).Scan(
		&userDTO.Email, &userDTO.FirstName, &userDTO.LastName, &userDTO.Password, &departmentID,
		&userDTO.Image, &userDTO.Theme, &userDTO.CredentialsNonExpired, &userDTO.AccountNonExpired,
		&userDTO.AccountNonLocked, &userDTO.Enabled,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("no user found with email %s", email)
		}
		return nil, err
	}

	// Если department_id не NULL, то выполняем запрос на получение департамента
	if departmentID.Valid {
		userDTO.Department = &models.Department{ID: int(departmentID.Int64)} // Присваиваем ID департаменту
		// Получаем данные департамента из базы
		departmentQuery := `SELECT id, name, short_name, color FROM department WHERE id = $1`
		err = s.DB.QueryRow(departmentQuery, userDTO.Department.ID).Scan(
			&userDTO.Department.ID, &userDTO.Department.Name, &userDTO.Department.ShortName, &userDTO.Department.Color,
		)
		if err != nil {
			return &models.Response{
				Success:           false,
				Message:           "Error fetching department data",
				Data:              nil,
				ErrorsDescription: err.Error(),
			}, err
		}
	}

	// 2. Далее продолжаем обрабатывать бронирования и роли
	// Получаем связанные бронирования
	bookingsQuery := `SELECT id, room_id, user_email, time FROM booking WHERE user_email = $1`
	rows, err := s.DB.Query(bookingsQuery, userDTO.Email)
	if err != nil {
		return &models.Response{
			Success:           false,
			Message:           "Error fetching booking data",
			Data:              nil,
			ErrorsDescription: err.Error(),
		}, err
	}
	defer rows.Close()

	// Обрабатываем бронирования
	for rows.Next() {
		var booking models.Booking
		if err := rows.Scan(&booking.ID, &booking.Room.ID, &booking.User.Email, &booking.Time); err != nil {
			return &models.Response{
				Success:           false,
				Message:           "Error reading booking data",
				Data:              nil,
				ErrorsDescription: err.Error(),
			}, err
		}
		booking.User = userDTO // Присваиваем пользователя для бронирования
		userDTO.Bookings = append(userDTO.Bookings, booking)
	}

	// Если бронирований нет, присваиваем пустой массив
	if len(userDTO.Bookings) == 0 {
		userDTO.Bookings = []models.Booking{}
	}

	// Получаем связанные роли
	rolesQuery := `SELECT id, authority FROM role WHERE user_email = $1`
	rows, err = s.DB.Query(rolesQuery, userDTO.Email)
	if err != nil {
		return &models.Response{
			Success:           false,
			Message:           "Error fetching role data",
			Data:              nil,
			ErrorsDescription: err.Error(),
		}, err
	}
	defer rows.Close()

	// Обрабатываем роли
	for rows.Next() {
		var role models.Role
		if err := rows.Scan(&role.ID, &role.Authority); err != nil {
			return &models.Response{
				Success:           false,
				Message:           "Error reading role data",
				Data:              nil,
				ErrorsDescription: err.Error(),
			}, err
		}
		role.User = &userDTO // Присваиваем пользователя для роли
		userDTO.Roles = append(userDTO.Roles, role)
	}

	// Если ролей нет, присваиваем пустой массив
	if len(userDTO.Roles) == 0 {
		userDTO.Roles = []models.Role{}
	}

	// Возвращаем успешный ответ с данными о пользователе
	response := &models.Response{
		Success:           true,
		Message:           "User data fetched successfully",
		Data:              map[string]models.UserResponse{"user": models.UserToUserResponse(userDTO)},
		ErrorsDescription: nil,
	}

	return response, nil
}

// Обновить пользователя, департамент, бронирования и роли
func (s *Service) UpdateUser(updatedUser models.UserDTO) (*models.Response, error) {
	// Начинаем транзакцию для атомарных изменений
	tx, err := s.DB.Begin()
	if err != nil {
		return &models.Response{
			Success:           false,
			Message:           "Failed to start transaction",
			ErrorsDescription: fmt.Sprintf("Transaction start error: %v", err),
		}, err
	}
	defer tx.Rollback() // Откатить изменения в случае ошибки

	// Обновляем информацию о пользователе
	query := `
		UPDATE users
		SET first_name = $1, last_name = $2, password = $3, department_id = $4, theme = $5, image = $6
		WHERE email = $7
		RETURNING email, first_name, last_name, password, department_id, image, theme, credentials_non_expired, account_non_expired, account_non_locked, enabled
	`
	var user models.UserDTO
	err = tx.QueryRow(query, updatedUser.FirstName, updatedUser.LastName, updatedUser.Password,
		updatedUser.Department.ID, updatedUser.Theme, updatedUser.Image, updatedUser.Email).Scan(
		&user.Email, &user.FirstName, &user.LastName, &user.Password, &user.Department.ID,
		&user.Image, &user.Theme, &user.CredentialsNonExpired, &user.AccountNonExpired,
		&user.AccountNonLocked, &user.Enabled,
	)
	if err != nil {
		return &models.Response{
			Success:           false,
			Message:           "Failed to update user",
			ErrorsDescription: fmt.Sprintf("Error updating user: %v", err),
		}, err
	}

	// Обновляем департамент
	_, err = tx.Exec(`UPDATE department SET name = $1, short_name = $2, color = $3 WHERE id = $4`,
		updatedUser.Department.Name, updatedUser.Department.ShortName, updatedUser.Department.Color, updatedUser.Department.ID)
	if err != nil {
		return &models.Response{
			Success:           false,
			Message:           "Failed to update department",
			ErrorsDescription: fmt.Sprintf("Error updating department: %v", err),
		}, err
	}

	// Обновляем бронирования
	_, err = tx.Exec(`DELETE FROM user_booking WHERE user_email = $1`, updatedUser.Email)
	if err != nil {
		return &models.Response{
			Success:           false,
			Message:           "Failed to delete old bookings",
			ErrorsDescription: fmt.Sprintf("Error deleting bookings: %v", err),
		}, err
	}

	for _, booking := range updatedUser.Bookings {
		_, err = tx.Exec(`INSERT INTO user_booking (user_email, booking_id) VALUES ($1, $2)`, updatedUser.Email, booking.ID)
		if err != nil {
			return &models.Response{
				Success:           false,
				Message:           "Failed to insert new booking",
				ErrorsDescription: fmt.Sprintf("Error inserting booking: %v", err),
			}, err
		}
	}

	// Обновляем роли
	_, err = tx.Exec(`DELETE FROM user_role WHERE user_email = $1`, updatedUser.Email)
	if err != nil {
		return &models.Response{
			Success:           false,
			Message:           "Failed to delete old roles",
			ErrorsDescription: fmt.Sprintf("Error deleting roles: %v", err),
		}, err
	}

	for _, role := range updatedUser.Roles {
		_, err = tx.Exec(`INSERT INTO user_role (user_email, role_id) VALUES ($1, $2)`, updatedUser.Email, role.ID)
		if err != nil {
			return &models.Response{
				Success:           false,
				Message:           "Failed to insert new role",
				ErrorsDescription: fmt.Sprintf("Error inserting new role: %v", err),
			}, err
		}
	}

	// Завершаем транзакцию
	err = tx.Commit()
	if err != nil {
		return &models.Response{
			Success:           false,
			Message:           "Failed to commit transaction",
			ErrorsDescription: fmt.Sprintf("Transaction commit error: %v", err),
		}, err
	}

	// Возвращаем успешный ответ с данными о пользователе
	return &models.Response{
		Success:           true,
		Message:           "User updated successfully",
		Data:              map[string]models.UserDTO{"user": user},
		ErrorsDescription: nil,
	}, nil
}

func (s *Service) GetUserImage(email string) (*models.Response, error) {
	// Запрос к базе данных для получения изображения пользователя
	query := `SELECT image FROM users WHERE email = $1` // Используем $1 для параметра в PostgreSQL
	var imageData sql.NullString                        // Изображение может быть NULL

	// Выполняем запрос
	err := s.DB.QueryRow(query, email).Scan(&imageData)
	if err != nil {
		// Если ошибка в запросе
		if errors.Is(err, sql.ErrNoRows) {
			return &models.Response{
				Success:           false,
				Message:           "User not found",
				Data:              nil,
				ErrorsDescription: "No user found with the specified email",
			}, err
		}
		return &models.Response{
			Success:           false,
			Message:           "Failed to get user image",
			Data:              nil,
			ErrorsDescription: err.Error(),
		}, err
	}

	// Проверка на наличие изображения
	if !imageData.Valid {
		return &models.Response{
			Success:           false,
			Message:           "Image not found",
			Data:              nil,
			ErrorsDescription: "User does not have an image",
		}, fmt.Errorf("user image not found")
	}

	// Чтение изображения с файловой системы
	imagePath := imageData.String // Путь к изображению
	imgData, err := os.ReadFile(imagePath)
	if err != nil {
		// Ошибка при чтении файла
		if os.IsNotExist(err) {
			return &models.Response{
				Success:           false,
				Message:           "Image file does not exist",
				Data:              nil,
				ErrorsDescription: "File not found at the specified path",
			}, err
		}
		return &models.Response{
			Success:           false,
			Message:           "Failed to read image file",
			Data:              nil,
			ErrorsDescription: err.Error(),
		}, err
	}

	// Возвращаем успешный ответ с изображением
	return &models.Response{
		Success:           true,
		Message:           "User image retrieved successfully",
		Data:              imgData, // Возвращаем изображение в виде []byte
		ErrorsDescription: nil,
	}, nil
}

// UpdateUserImage обновляет изображение пользователя
func (s *Service) UpdateUserImage(imageData []byte, email string) (*models.Response, error) {
	// Проверяем, существует ли пользователь
	var exists bool
	err := s.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`, email).Scan(&exists)
	if err != nil {
		return &models.Response{
			Success:           false,
			Message:           "Failed to check user existence",
			Data:              nil,
			ErrorsDescription: fmt.Sprintf("Error checking user existence: %v", err),
		}, err
	}

	if !exists {
		return &models.Response{
			Success:           false,
			Message:           "User not found",
			Data:              nil,
			ErrorsDescription: "No user found with the specified email",
		}, fmt.Errorf("user not found")
	}

	// Проверяем, что данные не пустые
	if len(imageData) == 0 {
		return &models.Response{
			Success:           false,
			Message:           "Invalid image data",
			Data:              nil,
			ErrorsDescription: "Image data is empty",
		}, fmt.Errorf("empty image data")
	}

	// Проверяем, является ли файл изображением
	_, format, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return &models.Response{
			Success:           false,
			Message:           "Invalid image format",
			Data:              nil,
			ErrorsDescription: "Uploaded file is not a valid image",
		}, fmt.Errorf("invalid image format")
	}

	// Разрешенные форматы (можно расширить)
	allowedFormats := map[string]bool{"jpeg": true, "png": true}
	if !allowedFormats[format] {
		return &models.Response{
			Success:           false,
			Message:           "Unsupported image format",
			Data:              nil,
			ErrorsDescription: fmt.Sprintf("Only JPEG and PNG formats are allowed, got: %s", format),
		}, fmt.Errorf("unsupported image format: %s", format)
	}

	// Формируем путь для изображения
	imagePath := fmt.Sprintf("images/%s.%s", email, format)

	// Создаем директорию, если её нет
	if err := os.MkdirAll("images", 0755); err != nil {
		return &models.Response{
			Success:           false,
			Message:           "Failed to create image directory",
			Data:              nil,
			ErrorsDescription: fmt.Sprintf("Error creating image directory: %v", err),
		}, err
	}

	// Записываем изображение на диск
	err = os.WriteFile(imagePath, imageData, 0644)
	if err != nil {
		return &models.Response{
			Success:           false,
			Message:           "Failed to write image to disk",
			Data:              nil,
			ErrorsDescription: fmt.Sprintf("Error writing image to disk: %v", err),
		}, err
	}

	// Обновляем путь к изображению в базе данных
	_, err = s.DB.Exec(`UPDATE users SET image = $1 WHERE email = $2`, imagePath, email)
	if err != nil {
		return &models.Response{
			Success:           false,
			Message:           "Failed to update image path in database",
			Data:              nil,
			ErrorsDescription: fmt.Sprintf("Error updating image path: %v", err),
		}, err
	}

	// Возвращаем успешный ответ
	return &models.Response{
		Success:           true,
		Message:           "Image updated successfully",
		Data:              map[string]string{"imagePath": imagePath},
		ErrorsDescription: nil,
	}, nil
}

func (s *Service) ChangePassword(oldPassword, newPassword, email string) (*models.Response, error) {
	// Получаем текущий хеш пароля из базы
	var hashedPassword string
	err := s.DB.QueryRow(`SELECT password FROM users WHERE email = $1`, email).Scan(&hashedPassword)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &models.Response{
				Success:           false,
				Message:           "User not found",
				Data:              nil,
				ErrorsDescription: "No user found with the specified email",
			}, fmt.Errorf("user not found")
		}
		return &models.Response{
			Success:           false,
			Message:           "Failed to fetch user password",
			Data:              nil,
			ErrorsDescription: fmt.Sprintf("Error fetching password: %v", err),
		}, err
	}

	// Проверяем старый пароль
	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(oldPassword))
	if err != nil {
		return &models.Response{
			Success:           false,
			Message:           "Incorrect old password",
			Data:              nil,
			ErrorsDescription: "Old password does not match",
		}, fmt.Errorf("incorrect old password")
	}

	// Проверяем сложность нового пароля (минимум 8 символов)
	if len(newPassword) < 8 {
		return &models.Response{
			Success:           false,
			Message:           "Password too short",
			Data:              nil,
			ErrorsDescription: "Password must be at least 8 characters long",
		}, fmt.Errorf("password too short")
	}

	// Хэшируем новый пароль
	newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return &models.Response{
			Success:           false,
			Message:           "Failed to hash new password",
			Data:              nil,
			ErrorsDescription: fmt.Sprintf("Error hashing password: %v", err),
		}, err
	}

	// Обновляем пароль в базе данных
	_, err = s.DB.Exec(`UPDATE users SET password = $1 WHERE email = $2`, string(newHashedPassword), email)
	if err != nil {
		return &models.Response{
			Success:           false,
			Message:           "Failed to update password",
			Data:              nil,
			ErrorsDescription: fmt.Sprintf("Error updating password: %v", err),
		}, err
	}

	// Возвращаем успешный ответ
	return &models.Response{
		Success:           true,
		Message:           "Password changed successfully",
		Data:              nil,
		ErrorsDescription: nil,
	}, nil
}

func (s *Service) DeleteUser(email string) (*models.Response, error) {
	// Открываем транзакцию
	tx, err := s.DB.Begin()
	if err != nil {
		return &models.Response{
			Success:           false,
			Message:           "Failed to start transaction",
			Data:              nil,
			ErrorsDescription: fmt.Sprintf("Error starting transaction: %v", err),
		}, err
	}
	// Если что-то пойдет не так — откатим изменения
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Получаем путь к изображению перед удалением пользователя
	var imagePath sql.NullString
	err = tx.QueryRow(`SELECT image FROM users WHERE email = $1`, email).Scan(&imagePath)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &models.Response{
				Success:           false,
				Message:           "User not found",
				Data:              nil,
				ErrorsDescription: "No user found with the specified email",
			}, fmt.Errorf("user not found")
		}
		return &models.Response{
			Success:           false,
			Message:           "Failed to fetch user data",
			Data:              nil,
			ErrorsDescription: fmt.Sprintf("Error fetching user data: %v", err),
		}, err
	}

	// Удаляем все связанные записи (бронирования, роли и т. д.)
	queries := []string{
		`DELETE FROM user_booking WHERE user_email = $1`,
		`DELETE FROM user_role WHERE user_email = $1`,
	}

	for _, query := range queries {
		_, err = tx.Exec(query, email)
		if err != nil {
			return &models.Response{
				Success:           false,
				Message:           "Failed to delete user data",
				Data:              nil,
				ErrorsDescription: fmt.Sprintf("Error executing query: %v", err),
			}, err
		}
	}

	// Удаляем пользователя
	_, err = tx.Exec(`DELETE FROM users WHERE email = $1`, email)
	if err != nil {
		return &models.Response{
			Success:           false,
			Message:           "Failed to delete user",
			Data:              nil,
			ErrorsDescription: fmt.Sprintf("Error deleting user: %v", err),
		}, err
	}

	// Удаляем изображение, если оно есть
	if imagePath.Valid {
		err = os.Remove(imagePath.String)
		if err != nil {
			log.Println("Failed to delete user image:", err)
		}
	}

	// Фиксируем изменения
	err = tx.Commit()
	if err != nil {
		return &models.Response{
			Success:           false,
			Message:           "Failed to commit transaction",
			Data:              nil,
			ErrorsDescription: fmt.Sprintf("Error committing transaction: %v", err),
		}, err
	}

	// Возвращаем успешный ответ
	return &models.Response{
		Success:           true,
		Message:           "User deleted successfully",
		Data:              nil,
		ErrorsDescription: nil,
	}, nil
}

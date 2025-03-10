package users

import (
	"book_talk/internal/auth"
	"book_talk/internal/models"
	"database/sql"
	"errors"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"log"
	"mime"
	"net/http"
	"os"
)

type Service struct {
	DB *sql.DB
}

func NewUsersService(db *sql.DB) *Service {
	return &Service{DB: db}
}

func (s *Service) GetAllUsers() (*models.Response, error) {
	query := `
		SELECT u.email, u.first_name, u.last_name, u.image, u.theme, 
			   u.credentials_non_expired, u.account_non_expired, 
			   u.account_non_locked, u.enabled, 
			   b.id, b.time, r.id, r.authority 
		FROM users u
		LEFT JOIN booking b ON u.email = b.user_email
		LEFT JOIN role ur ON u.email = ur.user_email
		LEFT JOIN role r ON ur.id = r.id
		ORDER BY u.email
	`

	rows, err := s.DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении пользователей: %v", err)
	}
	defer rows.Close()

	userMap := make(map[string]*models.UserResponse)

	for rows.Next() {
		var email, firstName, lastName, theme string
		var imageValue sql.NullString
		var credentialsNonExpired, accountNonExpired, accountNonLocked, enabled bool
		var bookingID, roleID *int
		var bookingDate, roleName *string

		if err := rows.Scan(&email, &firstName, &lastName, &imageValue, &theme,
			&credentialsNonExpired, &accountNonExpired,
			&accountNonLocked, &enabled,
			&bookingID, &bookingDate, &roleID, &roleName); err != nil {
			return nil, fmt.Errorf("ошибка при распознавании данных пользователя: %v", err)
		}

		user, exists := userMap[email]
		if !exists {
			user = &models.UserResponse{
				Email:                 email,
				FirstName:             firstName,
				LastName:              lastName,
				Theme:                 theme,
				Image:                 &imageValue.String,
				CredentialsNonExpired: credentialsNonExpired,
				AccountNonExpired:     accountNonExpired,
				AccountNonLocked:      accountNonLocked,
				Enabled:               enabled,
				Bookings:              []models.Booking{},
				Roles:                 []models.Role{},
			}
			userMap[email] = user
		}

		if bookingID != nil && bookingDate != nil {
			user.Bookings = append(user.Bookings, models.Booking{
				ID:   *bookingID,
				Time: *bookingDate,
			})
		}

		if roleID != nil && roleName != nil {
			user.Roles = append(user.Roles, models.Role{
				ID:        *roleID,
				Authority: *roleName,
			})
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка при обработке строк: %v", err)
	}

	var users []models.UserResponse
	for _, user := range userMap {
		users = append(users, *user)
	}

	return &models.Response{
		Message: "Пользователи успешно получены",
		Data:    users,
	}, nil
}

func (s *Service) GetUser(email string) (*models.Response, error) {
	var userDTO models.UserDTO
	var departmentID sql.NullInt64

	// Запрос для получения данных пользователя
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
			return nil, fmt.Errorf("пользователь с email %s не найден", email)
		}
		return nil, fmt.Errorf("ошибка при получении данных пользователя: %v", err)
	}

	// Получение данных о департаменте
	if departmentID.Valid {
		userDTO.Department = &models.Department{ID: int(departmentID.Int64)}
		departmentQuery := `SELECT id, name, short_name, color FROM department WHERE id = $1`
		err = s.DB.QueryRow(departmentQuery, userDTO.Department.ID).Scan(
			&userDTO.Department.ID, &userDTO.Department.Name, &userDTO.Department.ShortName, &userDTO.Department.Color,
		)
		if err != nil {
			return nil, fmt.Errorf("ошибка при получении данных департамента: %v", err)
		}
	}

	// Получение данных о бронированиях
	bookingsQuery := `SELECT id, room_id, user_email, time FROM booking WHERE user_email = $1`
	rows, err := s.DB.Query(bookingsQuery, userDTO.Email)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении данных о бронированиях: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var booking models.Booking
		if err := rows.Scan(&booking.ID, &booking.Room.ID, &booking.User.Email, &booking.Time); err != nil {
			return nil, fmt.Errorf("ошибка при обработке данных о бронированиях: %v", err)
		}
		booking.User = userDTO
		userDTO.Bookings = append(userDTO.Bookings, booking)
	}

	// Получение данных о ролях
	rolesQuery := `SELECT id, authority FROM role WHERE user_email = $1`
	rows, err = s.DB.Query(rolesQuery, userDTO.Email)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении данных о ролях: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var role models.Role
		if err := rows.Scan(&role.ID, &role.Authority); err != nil {
			return nil, fmt.Errorf("ошибка при обработке данных о ролях: %v", err)
		}
		role.User = &userDTO
		userDTO.Roles = append(userDTO.Roles, role)
	}

	return &models.Response{
		Message: "Данные о пользователе успешно получены",
		Data:    map[string]models.UserResponse{"user": models.UserToUserResponse(userDTO)},
	}, nil
}

func (s *Service) GetUserBookings(email string, page, size int) ([]models.Booking, error) {
	var bookings []models.Booking

	// Пагинация для бронирований
	offset := page * size
	bookingsQuery := `SELECT id, room_id, user_email, time FROM booking WHERE user_email = $1 LIMIT $2 OFFSET $3`
	rows, err := s.DB.Query(bookingsQuery, email, size, offset)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении данных о бронированиях: %v", err)
	}
	defer rows.Close()

	// Обрабатываем бронирования
	for rows.Next() {
		var booking models.Booking
		if err := rows.Scan(&booking.ID, &booking.Room.ID, &booking.User.Email, &booking.Time); err != nil {
			return nil, fmt.Errorf("ошибка при чтении данных о бронированиях: %v", err)
		}
		// Присваиваем объект пользователя (можно дополнительно извлечь его данные из базы, если нужно)
		booking.User = models.UserDTO{Email: email} // Простейшее присваивание
		bookings = append(bookings, booking)
	}

	// Если бронирований нет, возвращаем пустой срез
	if len(bookings) == 0 {
		bookings = []models.Booking{}
	}

	return bookings, nil
}

func (s *Service) UpdateUser(updatedUser models.UserDTO) (*models.UserDTO, error) {
	// Начинаем транзакцию для атомарных изменений
	tx, err := s.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("не удалось начать транзакцию: %v", err)
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
		return nil, fmt.Errorf("не удалось обновить пользователя: %v", err)
	}

	// Обновляем департамент
	_, err = tx.Exec(`UPDATE department SET name = $1, short_name = $2, color = $3 WHERE id = $4`,
		updatedUser.Department.Name, updatedUser.Department.ShortName, updatedUser.Department.Color, updatedUser.Department.ID)
	if err != nil {
		return nil, fmt.Errorf("не удалось обновить департамент: %v", err)
	}

	// Обновляем бронирования
	_, err = tx.Exec(`DELETE FROM user_booking WHERE user_email = $1`, updatedUser.Email)
	if err != nil {
		return nil, fmt.Errorf("не удалось удалить старые бронирования: %v", err)
	}

	for _, booking := range updatedUser.Bookings {
		_, err = tx.Exec(`INSERT INTO user_booking (user_email, booking_id) VALUES ($1, $2)`, updatedUser.Email, booking.ID)
		if err != nil {
			return nil, fmt.Errorf("не удалось вставить новое бронирование: %v", err)
		}
	}

	// Обновляем роли
	_, err = tx.Exec(`DELETE FROM user_role WHERE user_email = $1`, updatedUser.Email)
	if err != nil {
		return nil, fmt.Errorf("не удалось удалить старые роли: %v", err)
	}

	for _, role := range updatedUser.Roles {
		_, err = tx.Exec(`INSERT INTO user_role (user_email, role_id) VALUES ($1, $2)`, updatedUser.Email, role.ID)
		if err != nil {
			return nil, fmt.Errorf("не удалось вставить новую роль: %v", err)
		}
	}

	// Завершаем транзакцию
	err = tx.Commit()
	if err != nil {
		return nil, fmt.Errorf("не удалось подтвердить транзакцию: %v", err)
	}

	// Возвращаем обновленного пользователя
	return &user, nil
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
				Message: "User not found",
				Data:    nil,
			}, fmt.Errorf("user not found with email: %s", email)
		}
		return &models.Response{
			Message: "Failed to get user image",
			Data:    nil,
		}, fmt.Errorf("failed to get image from DB for user %s: %v", email, err)
	}

	// Проверка на наличие изображения
	if !imageData.Valid {
		return &models.Response{
			Message: "Image not found",
			Data:    nil,
		}, fmt.Errorf("no image found for user %s", email)
	}

	// Чтение изображения с файловой системы
	imagePath := imageData.String // Путь к изображению
	imgData, err := os.ReadFile(imagePath)
	if err != nil {
		// Ошибка при чтении файла
		if os.IsNotExist(err) {
			return &models.Response{
				Message: "Image file does not exist",
				Data:    nil,
			}, fmt.Errorf("image file does not exist at path: %s", imagePath)
		}
		return &models.Response{
			Message: "Failed to read image file",
			Data:    nil,
		}, fmt.Errorf("failed to read image file at %s: %v", imagePath, err)
	}

	// Возвращаем успешный ответ с изображением
	return &models.Response{
		Message: "User image retrieved successfully",
		Data:    imgData, // Возвращаем изображение в виде []byte
	}, nil
}

// UpdateUserImage обновляет изображение пользователя
func (s *Service) UpdateUserImage(imageData []byte, email string) (*models.Response, error) {
	// Проверяем, что данные не пустые
	if len(imageData) == 0 {
		return &models.Response{
			Message: "Invalid image data",
			Data:    nil,
		}, fmt.Errorf("empty image data")
	}

	// Проверка MIME-типа изображения
	contentType := http.DetectContentType(imageData)
	allowedFormats := map[string]bool{"image/jpeg": true, "image/png": true}
	if !allowedFormats[contentType] {
		return &models.Response{
			Message: "Unsupported image format",
			Data:    nil,
		}, fmt.Errorf("unsupported image format: %s", contentType)
	}

	// Формируем путь для изображения
	_, format := mime.ExtensionsByType(contentType)
	imagePath := fmt.Sprintf("images/%s%s", email, format)

	// Создаем директорию, если её нет
	if err := os.MkdirAll("images", 0755); err != nil {
		return &models.Response{
			Message: "Failed to create image directory",
			Data:    nil,
		}, err
	}

	// Записываем изображение на диск
	err := os.WriteFile(imagePath, imageData, 0644)
	if err != nil {
		return &models.Response{
			Message: "Failed to write image to disk",
			Data:    nil,
		}, err
	}

	// Обновляем изображение в базе данных и получаем путь к изображению, если пользователь существует
	var updatedEmail string
	err = s.DB.QueryRow(`
		UPDATE users 
		SET image = $1
		WHERE email = $2
		RETURNING email
	`, imagePath, email).Scan(&updatedEmail)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &models.Response{
				Message: "User not found",
				Data:    nil,
			}, err
		}

		return &models.Response{
			Message: "Failed to update image path in database",
			Data:    nil,
		}, err
	}

	// Возвращаем успешный ответ
	return &models.Response{
		Message: "Image updated successfully",
		Data:    map[string]string{"imagePath": imagePath},
	}, nil
}

func (s *Service) ChangePassword(oldPassword, newPassword, email string) (*models.Response, error) {
	// Получаем текущий хеш пароля из базы
	var hashedPassword string
	err := s.DB.QueryRow(`SELECT password FROM users WHERE email = $1`, email).Scan(&hashedPassword)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &models.Response{
				Message: "User not found",
				Data:    nil,
			}, fmt.Errorf("user not found")
		}
		return &models.Response{
			Message: "Failed to fetch user password",
			Data:    nil,
		}, err
	}

	// Проверяем старый пароль
	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(oldPassword))
	if err != nil {
		return &models.Response{
			Message: "Incorrect old password",
			Data:    nil,
		}, fmt.Errorf("incorrect old password")
	}

	isValid, errPass := auth.IsValidPassword(newPassword)
	if !isValid {
		return nil, fmt.Errorf("invalid password: %v", errPass)
	}

	// Хэшируем новый пароль
	newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return &models.Response{
			Message: "Failed to hash new password",
			Data:    nil,
		}, err
	}

	// Обновляем пароль в базе данных
	_, err = s.DB.Exec(`UPDATE users SET password = $1 WHERE email = $2`, string(newHashedPassword), email)
	if err != nil {
		return &models.Response{
			Message: "Failed to update password",
			Data:    nil,
		}, err
	}

	// Возвращаем успешный ответ
	return &models.Response{
		Message: "Password changed successfully",
		Data:    nil,
	}, nil
}

func (s *Service) DeleteUser(email string) (*models.Response, error) {
	// Открываем транзакцию
	tx, err := s.DB.Begin()
	if err != nil {
		return &models.Response{
			Message: "Не удалось начать транзакцию",
			Data:    nil,
		}, err
	}

	// Откатываем изменения, если ошибка произойдет
	defer func() {
		if err != nil {
			_ = tx.Rollback() // Игнорируем ошибку при откате, если она есть
		}
	}()

	// Получаем путь к изображению перед удалением пользователя
	var imagePath sql.NullString
	err = tx.QueryRow(`SELECT image FROM users WHERE email = $1`, email).Scan(&imagePath)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &models.Response{
				Message: "Пользователь не найден",
				Data:    nil,
			}, fmt.Errorf("пользователь не найден")
		}
		return &models.Response{
			Message: "Не удалось получить данные пользователя",
			Data:    nil,
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
				Message: "Не удалось удалить данные пользователя",
				Data:    nil,
			}, err
		}
	}

	// Удаляем пользователя
	_, err = tx.Exec(`DELETE FROM users WHERE email = $1`, email)
	if err != nil {
		return &models.Response{
			Message: "Не удалось удалить пользователя",
			Data:    nil,
		}, err
	}

	// Удаляем изображение, если оно есть
	if imagePath.Valid {
		err = os.Remove(imagePath.String)
		if err != nil {
			log.Println("Не удалось удалить изображение пользователя:", err)
		}
	}

	// Фиксируем изменения
	err = tx.Commit()
	if err != nil {
		return &models.Response{
			Message: "Не удалось зафиксировать транзакцию",
			Data:    nil,
		}, err
	}

	// Возвращаем успешный ответ
	return &models.Response{
		Message: "Пользователь успешно удален",
		Data:    nil,
	}, nil
}

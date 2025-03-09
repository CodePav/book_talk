package models

import "database/sql"

type UserDTO struct {
	Email                 string         `json:"email"`                 // Обязательное поле
	FirstName             string         `json:"firstName"`             // Обязательное поле
	LastName              string         `json:"lastName"`              // Обязательное поле
	Password              string         `json:"password"`              // Обязательное поле
	Department            *Department    `json:"department,omitempty"`  // Департамент, может быть nil
	Image                 sql.NullString `json:"image,omitempty"`       // Изображение, может быть nil
	Theme                 string         `json:"theme"`                 // Тема оформления
	Bookings              []Booking      `json:"bookings"`              // Список бронирований
	Roles                 []Role         `json:"roles"`                 // Роли пользователя
	CredentialsNonExpired bool           `json:"credentialsNonExpired"` // Флаг, указывающий, что учетные данные не истекли
	AccountNonExpired     bool           `json:"accountNonExpired"`     // Флаг, указывающий, что аккаунт не истек
	AccountNonLocked      bool           `json:"accountNonLocked"`      // Флаг, указывающий, что аккаунт не заблокирован
	Enabled               bool           `json:"enabled"`               // Флаг, указывающий, что аккаунт активен
}

func UserToUserResponse(user UserDTO) UserResponse {
	// Проверка, если изображение пустое или nil, не передаем его в ответ
	var image *string
	if user.Image.Valid && user.Image.String != "" {
		image = &user.Image.String
	}

	return UserResponse{
		Email:                 user.Email,
		FirstName:             user.FirstName,
		LastName:              user.LastName,
		Department:            user.Department,
		Image:                 image,
		Theme:                 user.Theme,
		Bookings:              user.Bookings,
		Roles:                 user.Roles,
		CredentialsNonExpired: user.CredentialsNonExpired,
		AccountNonExpired:     user.AccountNonExpired,
		AccountNonLocked:      user.AccountNonLocked,
		Enabled:               user.Enabled,
	}
}

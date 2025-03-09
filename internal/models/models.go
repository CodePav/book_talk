package models

import (
	"time"
)

type Response struct {
	Success           bool        `json:"success"`
	Message           string      `json:"message"`
	Data              interface{} `json:"data"`
	ErrorsDescription interface{} `json:"errorsDescription"`
}

type Address struct {
	ID       int    `json:"id"`
	Region   string `json:"region"`   // Область
	City     string `json:"city"`     // Город
	Street   string `json:"street"`   // Улица
	Building string `json:"building"` // Здание
}

type Booking struct {
	ID   int    `json:"id"`
	Room Room   `json:"room"` // Связь с комнатой
	User User   `json:"user"` // Связь с пользователем
	Time string `json:"time"` // Время бронирования в формате "YYYY-MM-DDTHH:MM:SSZ"
}

type Department struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	ShortName string `json:"shortName"`
	Color     string `json:"color"`
	Users     []User `json:"users"` // Связь с пользователями
}

type Role struct {
	ID        int    `json:"id"`
	Authority string `json:"authority"`
	User      *User  `json:"user"` // nullable Связь с пользователем
}

type Room struct {
	ID        int            `json:"id"`
	Capacity  int            `json:"capacity"`
	Name      string         `json:"name"`
	Address   Address        `json:"address"` // Связь с адресом
	ImagePath string         `json:"imagePath"`
	Weekdays  []time.Weekday `json:"weekdays"` // Связь с днями недели
	Active    bool           `json:"active"`
}

type LocalTime struct {
	Hour   int `json:"hour"`
	Minute int `json:"minute"`
	Second int `json:"second"`
}

type Weekday struct {
	ID        int    `json:"id"`
	Day       string `json:"day"`
	StartTime string `json:"startTime"` // Время начала, например, "09:00"
	EndTime   string `json:"endTime"`   // Время конца, например, "18:00"
	Room      Room   `json:"room"`      // Связь с комнатой
	Active    bool   `json:"active"`
}

type ShortUserInfo struct {
	Email     string `json:"email"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

type User struct {
	Email                 string      `json:"email"`                 // Обязательное поле
	FirstName             string      `json:"firstName"`             // Обязательное поле
	LastName              string      `json:"lastName"`              // Обязательное поле
	Password              string      `json:"password"`              // Обязательное поле
	Department            *Department `json:"department"`            // Департамент (обязательное)
	Image                 string      `json:"image,omitempty"`       // Изображение (опциональное)
	Theme                 string      `json:"theme"`                 // Тема оформления (обязательное поле)
	Bookings              []Booking   `json:"bookings"`              // Список бронирований
	Roles                 []Role      `json:"roles"`                 // Роли пользователя
	CredentialsNonExpired bool        `json:"credentialsNonExpired"` // Флаг, указывающий, что учетные данные не истекли
	AccountNonExpired     bool        `json:"accountNonExpired"`     // Флаг, указывающий, что аккаунт не истек
	AccountNonLocked      bool        `json:"accountNonLocked"`      // Флаг, указывающий, что аккаунт не заблокирован
	Enabled               bool        `json:"enabled"`               // Флаг, указывающий, что аккаунт активен
}

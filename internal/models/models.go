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

type Pagination struct {
	Page int `json:"page"`
	Size int `json:"size"`
}

type Address struct {
	ID       int    `json:"id"`
	Region   string `json:"region"`   // Область
	City     string `json:"city"`     // Город
	Street   string `json:"street"`   // Улица
	Building string `json:"building"` // Здание
}

type Booking struct {
	ID   int     `json:"id"`
	Room Room    `json:"room"` // Связь с комнатой
	User UserDTO `json:"user"` // Связь с пользователем
	Time string  `json:"time"` // Время бронирования в формате "YYYY-MM-DDTHH:MM:SSZ"
}

type Department struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	ShortName string    `json:"shortName"`
	Color     string    `json:"color"`
	Users     []UserDTO `json:"users"` // Связь с пользователями
}

type Role struct {
	ID        int      `json:"id"`
	Authority string   `json:"authority"`
	User      *UserDTO `json:"user"` // nullable Связь с пользователем
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

type ShortUserResponse struct {
	Email     string `json:"email"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

type UserResponse struct {
	Email                 string      `json:"email"`
	FirstName             string      `json:"firstName"`
	LastName              string      `json:"lastName"`
	Department            *Department `json:"department"`
	Image                 *string     `json:"image"`
	Theme                 string      `json:"theme"`
	Bookings              []Booking   `json:"bookings"`
	Roles                 []Role      `json:"roles"`
	CredentialsNonExpired bool        `json:"credentialsNonExpired"`
	AccountNonExpired     bool        `json:"accountNonExpired"`
	AccountNonLocked      bool        `json:"accountNonLocked"`
	Enabled               bool        `json:"enabled"`
}

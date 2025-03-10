package models

import (
	"time"
)

// Response represents a standard API response with a success flag, message, data, and error description.
type Response struct {
	Message string      `json:"message"` // A message associated with the response
	Data    interface{} `json:"data"`    // The main data returned in the response
}

// Pagination represents pagination information for a list of results.
type Pagination struct {
	Page int `json:"page"` // Current page number
	Size int `json:"size"` // Number of items per page
}

// Address represents an address with fields like region, city, street, and building.
type Address struct {
	ID       int    `json:"id"`       // Unique identifier for the address
	Region   string `json:"region"`   // Region or area of the address (e.g., state or province)
	City     string `json:"city"`     // City of the address
	Street   string `json:"street"`   // Street name of the address
	Building string `json:"building"` // Specific building or structure at the address
}

// Booking represents a room booking with details about the room, user, and time of booking.
type Booking struct {
	ID   int     `json:"id"`   // Unique identifier for the booking
	Room Room    `json:"room"` // Room being booked (reference to Room struct)
	User UserDTO `json:"user"` // User who made the booking (reference to UserDTO struct)
	Time string  `json:"time"` // Booking time in "YYYY-MM-DDTHH:MM:SSZ" format
}

// Department represents a department with a list of associated users and other details.
type Department struct {
	ID        int       `json:"id"`        // Unique identifier for the department
	Name      string    `json:"name"`      // Full name of the department
	ShortName string    `json:"shortName"` // Abbreviated name for the department
	Color     string    `json:"color"`     // Color associated with the department (e.g., for UI display)
	Users     []UserDTO `json:"users"`     // List of users in this department (references UserDTO structs)
}

// Role represents a user role with associated authority and an optional link to the user.
type Role struct {
	ID        int      `json:"id"`        // Unique identifier for the role
	Authority string   `json:"authority"` // Authority or permission granted by the role
	User      *UserDTO `json:"user"`      // Optional link to the user with this role (nullable)
}

// Room represents a room with details such as capacity, name, address, image, weekdays, and active status.
type Room struct {
	ID        int            `json:"id"`        // Unique identifier for the room
	Capacity  int            `json:"capacity"`  // Maximum capacity of the room
	Name      string         `json:"name"`      // Name of the room
	Address   Address        `json:"address"`   // The address of the room (reference to Address struct)
	ImagePath string         `json:"imagePath"` // Path to an image of the room
	Weekdays  []time.Weekday `json:"weekdays"`  // List of weekdays when the room is available
	Active    bool           `json:"active"`    // Indicates if the room is currently active
}

// LocalTime represents a specific time with hour, minute, and second components.
type LocalTime struct {
	Hour   int `json:"hour"`   // Hour component of the time
	Minute int `json:"minute"` // Minute component of the time
	Second int `json:"second"` // Second component of the time
}

// Weekday represents the availability of a room on a specific day with start and end times.
type Weekday struct {
	ID        int    `json:"id"`        // Unique identifier for the weekday availability
	Day       string `json:"day"`       // Day of the week (e.g., Monday, Tuesday)
	StartTime string `json:"startTime"` // Start time in "HH:MM" format
	EndTime   string `json:"endTime"`   // End time in "HH:MM" format
	Room      Room   `json:"room"`      // The room being described (reference to Room struct)
	Active    bool   `json:"active"`    // Indicates if the room is available on this day
}

// ShortUserResponse is a simplified version of a user response, typically used in smaller data sets or summaries.
type ShortUserResponse struct {
	Email     string `json:"email"`     // User's email address
	FirstName string `json:"firstName"` // User's first name
	LastName  string `json:"lastName"`  // User's last name
}

// UserResponse represents detailed information about a user, including their department, roles, and bookings.
type UserResponse struct {
	Email                 string      `json:"email"`                 // User's email address
	FirstName             string      `json:"firstName"`             // User's first name
	LastName              string      `json:"lastName"`              // User's last name
	Department            *Department `json:"department"`            // Department the user belongs to (nullable)
	Image                 *string     `json:"image"`                 // Optional image URL (nullable)
	Theme                 string      `json:"theme"`                 // User's theme preference (e.g., light or dark mode)
	Bookings              []Booking   `json:"bookings"`              // List of bookings made by the user
	Roles                 []Role      `json:"roles"`                 // List of roles assigned to the user
	CredentialsNonExpired bool        `json:"credentialsNonExpired"` // Whether the user's credentials are expired
	AccountNonExpired     bool        `json:"accountNonExpired"`     // Whether the user's account is expired
	AccountNonLocked      bool        `json:"accountNonLocked"`      // Whether the user's account is locked
	Enabled               bool        `json:"enabled"`               // Whether the user is enabled (active)
}

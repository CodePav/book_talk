package main

import (
	"github.com/gorilla/mux"
	"log"
	"net/http"
)

func main() {
	r := mux.NewRouter()

	// Применение аутентификации ко всем маршрутам
	//r.Use(middleware.AuthenticationMiddleware)

	// Регистрация маршрутов для секций с авторизацией (например, только администраторы могут создавать секции)
	//r.Handle("/sections", middleware.AuthorizationMiddleware("admin", http.HandlerFunc(sections.CreateSection))).Methods("POST")
	//r.Handle("/sections", http.HandlerFunc(sections.GetAllSections)).Methods("GET")

	// Регистрация маршрутов для комнат
	//r.Handle("/rooms", http.HandlerFunc(rooms.GetAllRooms)).Methods("GET")
	//r.Handle("/rooms", middleware.AuthorizationMiddleware("admin", http.HandlerFunc(rooms.CreateRoom))).Methods("POST")

	// Запуск сервера
	log.Fatal(http.ListenAndServe(":8080", r))
}

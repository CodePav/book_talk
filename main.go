package main

import (
	"book_talk/internal/auth"
	"book_talk/internal/database"
	"book_talk/internal/users"
	"book_talk/middleware"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {
	database, err := db.ConnectDB()
	if err != nil {
		log.Fatal("Ошибка подключения к БД:", err)
	}
	defer database.Close()

	authHandler := auth.NewAuthHandler(database)
	usersHandler := users.NewUsersHandler(database)

	// Создаем основной роутер
	r := mux.NewRouter()

	// Группа маршрутов для аутентификации
	authRouter := r.PathPrefix("/api/v1/auth").Subrouter()
	authRouter.HandleFunc("/signup", authHandler.Register).Methods("POST")
	authRouter.HandleFunc("/login", authHandler.Login).Methods("POST")
	authRouter.HandleFunc("/refresh", authHandler.Refresh).Methods("GET")

	// Группа маршрутов для пользователей
	usersRouter := r.PathPrefix("/api/v1").Subrouter()
	usersRouter.HandleFunc("/me", mw.Protect(usersHandler.GetCurrentUser)).Methods("GET")
	usersRouter.HandleFunc("/me", mw.Protect(usersHandler.UpdateUser)).Methods("PUT")
	usersRouter.HandleFunc("/me", mw.Protect(usersHandler.DeleteUser)).Methods("DELETE")
	usersRouter.HandleFunc("/me/image", mw.Protect(usersHandler.GetUserImage)).Methods("GET")
	usersRouter.HandleFunc("/me/image", mw.Protect(usersHandler.UpdateUserImage)).Methods("PUT")
	usersRouter.HandleFunc("/me/change-password", mw.Protect(usersHandler.ChangePassword)).Methods("PUT")
	usersRouter.HandleFunc("/users", mw.Protect(usersHandler.GetAllUsers)).Methods("GET")

	// Запуск сервера
	log.Println("Сервер запущен на порту 8080...")
	log.Fatal(http.ListenAndServe(":8080", r))

}

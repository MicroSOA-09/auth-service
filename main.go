package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/MicroSOA-09/auth-service/handler"
	"github.com/MicroSOA-09/auth-service/repository"
	"github.com/MicroSOA-09/auth-service/service"
	gorillaHandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	// Učitaj .env fajl
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file:", err)
	}

	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = "8080"
	}

	timeoutContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger := log.New(os.Stdout, "[auth-handler] ", log.LstdFlags)
	storeLogger := log.New(os.Stdout, "[user-repo] ", log.LstdFlags)

	userRepo, err := repository.New(timeoutContext, storeLogger)
	if err != nil {
		logger.Fatal(err)
	}
	defer userRepo.Disconnect(timeoutContext)

	jwt_secret := os.Getenv("JWT_SECRET")
	mailUser := os.Getenv("MAIL_USER")
	mailAppPassword := os.Getenv("MAIL_APP_PASSWORD")
	if mailUser == "" || mailAppPassword == "" {
		logger.Fatal("MAIL_USER and MAIL_APP_PASSWORD are required")
	}
	emailClient := service.NewEmailClient("smtp.gmail.com", 587, mailUser, mailAppPassword, mailUser, logger)

	authService := service.NewAuthService(userRepo, jwt_secret, emailClient)
	authHandler := handler.NewAuthHandler(authService, logger)
	userService := service.NewUserService(userRepo)
	userHandler := handler.NewUserHandler(userService, logger)

	router := mux.NewRouter()
	router.Use(authHandler.MiddlewareContentTypeSet)

	// AUTH ROUTES
	authRouter := router.Methods(http.MethodPost).Subrouter()
	authRouter.HandleFunc("/api/auth/register", authHandler.Register)
	authRouter.HandleFunc("/api/auth/login", authHandler.Login)
	// confirm mail
	// password reset

	// USER ROUTES
	getRouter := router.Methods(http.MethodGet).Subrouter()
	getRouter.HandleFunc("/api/user", userHandler.GetAll)

	cors := gorillaHandlers.CORS(gorillaHandlers.AllowedOrigins([]string{"*"}))

	//Initialize the server
	server := http.Server{
		Addr:         ":" + port,
		Handler:      cors(router),
		IdleTimeout:  120 * time.Second,
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 1 * time.Second,
	}

	logger.Println("Server listening on port", port)
	//Distribute all the connections to goroutines
	go func() {
		err := server.ListenAndServe()
		if err != nil {
			logger.Fatal(err)
		}
	}()

	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, os.Interrupt)
	signal.Notify(sigCh, os.Kill)

	sig := <-sigCh
	logger.Println("Received terminate, graceful shutdown", sig)

	//Try to shutdown gracefully
	if server.Shutdown(timeoutContext) != nil {
		logger.Fatal("Cannot gracefully shutdown...")
	}
	logger.Println("Server stopped")

}

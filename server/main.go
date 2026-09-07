package main

import (
	"encoding/json"
	"log"
	"net/http"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"strings"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}


type User struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	Role         string `json:"role"`
	PasswordHash string `json:"-"`
}


var users map[string]User


func logoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("sid")
	if err == nil {
		// Eliminar la sesión de Redis
		rdb.Del(ctx, "sess:"+cookie.Value)
	}

	// Eliminar la cookie del navegador
	http.SetCookie(w, &http.Cookie{
		Name:     "sid",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("logout ok"))
}

func profileHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authHeader := r.Header.Get("Authorization")

	if authHeader == "" {
		http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
		return
	}

	const prefix = "Bearer "

	if !strings.HasPrefix(authHeader, prefix) {
		http.Error(w, "Invalid Authorization header", http.StatusUnauthorized)
		return
	}

	tokenString := strings.TrimPrefix(authHeader, prefix)

	token, err := VerifyJWT(tokenString)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	fmt.Println("JWT valid:", token.Valid)

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		http.Error(w, "Invalid claims", http.StatusUnauthorized)
		return
	}

	userID := claims["user_id"]
	role := claims["role"]

	fmt.Println("User ID:", userID)
	fmt.Println("Role:", role)

	user, exists := users[userID]
	
	if !exists { 
		http.Error(w, "Invalid userID", http.StatusUnauthorized) 
		return 
	}

	response := map[string]string{
             	"id":	    user.ID,
               	"username": user.Username,
                "role":     user.Role,
        }

        json.NewEncoder(w).Encode(response)

}



func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Printf("Method: %s\n", r.Method)
	fmt.Printf("URL: %s\n", r.URL)
	fmt.Printf("Headers: %v\n", r.Header)
	response := map[string]string{
		"status": "ok",
	}

	json.NewEncoder(w).Encode(response)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Only POST is allowed
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Decode JSON request body
	var login LoginRequest

	if err := json.NewDecoder(r.Body).Decode(&login); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Temporary hardcoded credentials.
	// We will replace this later with proper authentication.
	user, exists := users[login.Username]

	if !exists || !CheckPassword(user.PasswordHash, login.Password) {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}
	
	// Create JWT.
	tokenString, err := CreateJWT(user)
	if err != nil {
		http.Error(w, "Error creating token", http.StatusInternalServerError)
		return
	}


	response := map[string]string{
		"token": tokenString,
	}

	json.NewEncoder(w).Encode(response)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "server/index.html")
}

func jwtTestHandler(w http.ResponseWriter, r *http.Request) {
	user := User{
		ID:   "alice",
		Role: "user",
	}

	tokenString, err := CreateJWT(user)
	if err != nil {
		http.Error(w, "error creating token", http.StatusInternalServerError)
		return
	}

	token, err := VerifyJWT(tokenString)
	if err != nil {
		http.Error(w, "invalid token: "+err.Error(), http.StatusUnauthorized)
		return
	}

	fmt.Fprintf(w, "JWT válido\n")
	fmt.Fprintf(w, "Token: %s\n", tokenString)
	fmt.Fprintf(w, "Valid: %v\n", token.Valid)
}

func main() {
	hash, err := HashPassword("secret123")
	
	if err != nil {
    		log.Fatal(err)
	}

	log.Println("Hash:", hash)
	log.Println("Correct:", CheckPassword(hash, "secret123"))
	log.Println("Wrong:", CheckPassword(hash, "wrong"))
	fmt.Println("Salt:", hash[7:29])

	users = map[string]User{
		"alice": {
			ID:           "123",
			Username:     "alice",
			Role:         "admin",
			PasswordHash: hash,
		},
	}

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/login", loginHandler)	
	http.HandleFunc("/profile", profileHandler)
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/logout", logoutHandler)
	http.HandleFunc("/jwt-test", jwtTestHandler)

	log.Println("Server listening on :8080")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

package main

import (
	"encoding/json"
	"log"
	"net/http"
	"fmt"
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
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cookie, err := r.Cookie("sid")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	user, exists := getRedisSession(cookie.Value)
	if !exists {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	response := map[string]string{
		"id":       user.ID,
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
	
	sessionID, err := createRedisSession(user)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	
	//sessionID := createMemorySession(user)

	http.SetCookie(w, &http.Cookie{
		Name:     "sid",
		Value:    sessionID,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	})

	response := map[string]string{
		"message": "login ok",
	}

	json.NewEncoder(w).Encode(response)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "server/index.html")
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

	log.Println("Server listening on :8080")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

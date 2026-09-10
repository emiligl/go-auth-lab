package main

import (
	"context"
	"time"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
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

type contextKey string

const (
	userNameKey contextKey = "userName"
	jtiKey      contextKey = "jti"
	expKey      contextKey = "exp"
)

var redisClient = redis.NewClient(&redis.Options{
	Addr: "localhost:6379",
})

func main() {
	
	ctx := context.Background()

	_, err := redisClient.Ping(ctx).Result()

	if err != nil {
		fmt.Println("Redis connection error:", err)
		return
	}

	fmt.Println("Redis connected")

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

	http.HandleFunc("/login", loginHandler)	

	http.Handle(
		"/profile",
		jwtMiddleware(http.HandlerFunc(profileHandler)),
	)

	http.Handle(
		"/logout",
		jwtMiddleware(http.HandlerFunc(logoutHandler)),
	)

	http.HandleFunc("/", indexHandler)

	log.Println("Server listening on :8080")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}

}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "server/index.html")
}

func loginHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var login LoginRequest

	err := json.NewDecoder(r.Body).Decode(&login)

	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	user, exists := users[login.Username]

	if !exists || !CheckPassword(user.PasswordHash, login.Password) {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	tokenString, err := CreateJWT(user)

	if err != nil {
		http.Error(w, "Error creating token", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    tokenString,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})

	w.Header().Set("Content-Type", "application/json")

	response := map[string]string{
		"message": "Login successful",
	}

	json.NewEncoder(w).Encode(response)
}

func jwtMiddleware(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		cookie, err := r.Cookie("access_token")

		if err != nil {
			http.Error(w, "Missing access token", http.StatusUnauthorized)
			return
		}

		tokenString := cookie.Value

		token, err := VerifyJWT(tokenString)

		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		if !token.Valid {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)

		if !ok {
			http.Error(w, "Invalid claims", http.StatusUnauthorized)
			return
		}

		userName, ok := claims["user_name"].(string)

		if !ok {
			http.Error(w, "Invalid username", http.StatusUnauthorized)
			return
		}
		
		jti, ok := claims["jti"].(string)

		if !ok {
			http.Error(w, "Invalid token ID", http.StatusUnauthorized)
			return
		}

		exp, ok := claims["exp"].(float64)

		if !ok {
			http.Error(w, "Invalid expiration", http.StatusUnauthorized)
			return
		}


		expiresAt := time.Unix(int64(exp), 0)


		revoked, err := redisClient.Exists(
			context.Background(),
			"revoked:"+jti,
		).Result()

		if err != nil {
			http.Error(w, "Redis error", http.StatusInternalServerError)
			return
		}

		if revoked > 0 {
			http.Error(w, "Token revoked", http.StatusUnauthorized)
			return
		}


		ctx := context.WithValue(
			r.Context(),
			userNameKey,
			userName,
		)

		ctx = context.WithValue(
			ctx,
			jtiKey,
			jti,
		)

		ctx = context.WithValue(
			ctx,
			expKey,
			expiresAt,
		)

		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

func profileHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userName, ok := r.Context().Value(userNameKey).(string)

	if !ok {
		http.Error(w, "User not found in context", http.StatusUnauthorized)
		return
	}

	user, exists := users[userName]

	if !exists {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	response := map[string]string{
		"id":       user.ID,
		"username": user.Username,
		"role":     user.Role,
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(response)
}
	
func logoutHandler(w http.ResponseWriter, r *http.Request) {

	jti, ok := r.Context().Value(jtiKey).(string)

	if !ok {
		http.Error(w, "JTI not found", http.StatusUnauthorized)
		return
	}

	expiresAt, ok := r.Context().Value(expKey).(time.Time)

	if !ok {
		http.Error(w, "Expiration not found", http.StatusUnauthorized)
		return
	}

	ttl := time.Until(expiresAt)

	if ttl <= 0 {
		http.Error(w, "Token already expired", http.StatusUnauthorized)
		return
	}

	ctx := context.Background()

	err := redisClient.Set(
		ctx,
		"revoked:"+jti,
		"true",
		ttl,
	).Err()

	if err != nil {
		http.Error(w, "Redis error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
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

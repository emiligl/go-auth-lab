package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func health(client *http.Client) {
	req, err := http.NewRequest(
		http.MethodGet,
		"http://localhost:8080/health",
		nil,
	)
	if err != nil {
		panic(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	fmt.Println("Status:", resp.Status)
	fmt.Println("Response:", string(body))
}

func login(client *http.Client) {
	loginRequest := LoginRequest{
		Username: "alice",
		Password: "secret123",
	}

	body, err := json.Marshal(loginRequest)
	if err != nil {
		panic(err)
	}

	req, err := http.NewRequest(
		http.MethodPost,
		"http://localhost:8080/login",
		bytes.NewBuffer(body),
	)
	if err != nil {
		panic(err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	fmt.Println("Status:", resp.Status)
	fmt.Println("Response:", string(responseBody))
}

func profile(client *http.Client) {
	req, err := http.NewRequest(
		http.MethodGet,
		"http://localhost:8080/profile",
		nil,
	)
	if err != nil {
		panic(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	fmt.Println("Status:", resp.Status)
	fmt.Println("Response:", string(body))
}

func main() {
	jar, err := cookiejar.New(nil)
	if err != nil {
		panic(err)
	}

	client := &http.Client{
		Jar: jar,
	}	


	fmt.Println("=== HEALTH ===")
	health(client)

	fmt.Println("\n=== LOGIN ===")
	login(client)

	u, _ := url.Parse("http://localhost:8080")
	fmt.Println("Cookies:", client.Jar.Cookies(u))

	fmt.Println("\n=== PROFILE ===")
	profile(client)

}

package main

import (
	"fmt"
	"log"
	"net/http"

	"tick/be/internal/auth"
)

func main() {
	http.HandleFunc("/", auth.HandleHome)
	http.HandleFunc("/auth/google/login", auth.HandleGoogleLogin)
	http.HandleFunc("/auth/google/callback", auth.HandleGoogleCallback)

	fmt.Println("Server started at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

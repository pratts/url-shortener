package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"shortener/models"
	"shortener/services"
)

func Init() {
	services.InitMap()
}

func InitUrls() {
	http.HandleFunc("/", urlHandler)

	fmt.Println("Server started at http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}

func urlHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("urlHandler called with path:", r.URL.Path, " method:", r.Method)
	if r.Method == http.MethodGet {
		expandUrl(w, r)
	} else if r.Method == http.MethodPost {
		shortenURL(w, r)
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func shortenURL(w http.ResponseWriter, r *http.Request) {
	fmt.Println("ShortenURL called")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body := r.Body
	defer body.Close()

	urlInput := models.UrlInput{}
	err := json.NewDecoder(body).Decode(&urlInput)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if urlInput.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}
	response := services.Shorten(urlInput.URL)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
	w.Write(nil)
}

func expandUrl(w http.ResponseWriter, r *http.Request) {
	fmt.Println("decodeUrl called")
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	fmt.Println("path: ", r.URL.Path)
	code := r.URL.Path[len("/"):] // Extract the code from the URL path
	fmt.Println("code: ", code)
	originalURL, err := services.Expand(code) // Call ExpandUrl with the extracted code
	fmt.Println("originalURL: ", originalURL)
	fmt.Println("err: ", err)
	if err != nil {
		http.Error(w, "URL not found", http.StatusNotFound)
		return
	}
	fmt.Println("Redirecting to original URL: ", originalURL)
	http.Redirect(w, r, originalURL, http.StatusFound)
}

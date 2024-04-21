package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

func main() {
	// Create a new router
	r := mux.NewRouter()

	// Define your routes
	r.HandleFunc("/login", loginHandler).Methods("GET")
	r.HandleFunc("/modify", modifyHandler).Methods("POST")

	// CORS setup
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"}, // Allow all origins
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	// Use the CORS middleware with the router
	handler := c.Handler(r)

	// Start the server
	port := ":3000"
	fmt.Printf("Server is running on http://localhost%s\n", port)
	http.ListenAndServe(port, handler)
}

var sessionCookie = ""

func loginHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Code parameter is required", http.StatusBadRequest)
		return
	}

	targetURL := "https://www.simplyhired.ca/api/auth/callback"
	queryParams := url.Values{}
	queryParams.Set("state", "eyJjc3JmVG9rZW4iOiJjb3Vjb3UiLCJyZWRpcmVjdFVybCI6Imh0dHBzOi8vd3d3LnNpbXBseWhpcmVkLmNhLyIsInJlZGlyZWN0RG9tYWluIjoiaHR0cHM6Ly93d3cuc2ltcGx5aGlyZWQuY2EifQ==")
	queryParams.Set("code", code)
	targetURL += "?" + queryParams.Encode()

	// Create a custom HTTP client with proxy settings and redirect policy
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Do not follow redirects
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	req.Header.Set("Cookie", "csrf=coucou;")
	req.Host = "www.simplyhired.ca"

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	statusCode := resp.StatusCode
	fmt.Println("Status code:", statusCode)

	// Extract cookies with the name 'session-cookie'
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "session-cookie" {
			sessionCookie = cookie.String()
			break
		}
	}

	// Make a request to the profile.json endpoint with the session cookie
	profileURL := "https://www.simplyhired.ca/_next/data/CrjGLb7NbUhm7ymGsRfbc/en-CA/profile.json"
	reqProfile, err := http.NewRequest("GET", profileURL, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	reqProfile.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36")
	reqProfile.Header.Set("Sec-Fetch-Site", "cross-site")
	reqProfile.Header.Set("Cookie", sessionCookie)

	respProfile, err := client.Do(reqProfile)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer respProfile.Body.Close()

	// Decode the JSON response
	var profileData map[string]interface{}
	err = json.NewDecoder(respProfile.Body).Decode(&profileData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	pageProps, ok := profileData["pageProps"].(map[string]interface{})
	if !ok {
		http.Error(w, "Failed to extract pageProps", http.StatusInternalServerError)
		return
	}

	mappedProfile, ok := pageProps["mappedProfile"].(map[string]interface{})
	if !ok {
		http.Error(w, "Failed to extract mappedProfile", http.StatusInternalServerError)
		return
	}

	// Send mappedProfile as JSON response
	jsonResponse, err := json.Marshal(mappedProfile)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonResponse)
}

func modifyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Decode the JSON request body
	var requestBody map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		http.Error(w, "Failed to decode JSON request body", http.StatusBadRequest)
		return
	}

	// Marshal the decoded request body to JSON
	requestBodyJSON, err := json.Marshal(requestBody)
	if err != nil {
		http.Error(w, "Failed to marshal request body to JSON", http.StatusInternalServerError)
		return
	}

	// Create a new HTTP client with custom headers
	client := &http.Client{}
	fmt.Printf(sessionCookie)

	// Create a new request
	req, err := http.NewRequest("POST", "https://www.simplyhired.ca/api/next/profile/update", bytes.NewReader(requestBodyJSON))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Add headers to the request
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	req.Header.Set("Cookie", sessionCookie)
	// Perform the request
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Send the response code
	w.Write([]byte(strconv.Itoa(resp.StatusCode)))
}

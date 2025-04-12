package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type Route struct {
	UrlStartsWith  string `json:"urlStartsWith"`
	RedirectHost   string `json:"redirectHost"`
	StripUrlPrefix string `json:"stripUrlPrefix"`
}

func parseRoutes(filename string) ([]Route, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var routes []Route
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&routes); err != nil {
		return nil, err
	}

	return routes, nil
}

func main() {

	routes, err := parseRoutes("C:\\Users\\gopi\\code\\github\\toy-reverse-proxy-go\\routes.json")
	if err != nil {
		fmt.Println("Error parsing routes:", err)
		return
	}
	fmt.Println("Parsed routes:", routes)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Define the backend server URL
		backendURL := "http://localhost:5173"
		path := r.URL.Path
		// Strip the "/console" prefix from the URL path
		if len(path) >= 8 && path[:8] == "/console" {
			path = path[8:]
		}
		// Create a new request to the backend server
		fmt.Println("Forwarding request to backend server:", backendURL+path)
		req, err := http.NewRequest(r.Method, backendURL+path, r.Body)
		if err != nil {
			http.Error(w, "Failed to create request", http.StatusInternalServerError)
			return
		}

		// Copy headers from the original request
		for key, values := range r.Header {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}

		// Perform the request to the backend server
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, "Failed to reach backend server", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		// Copy the response from the backend server to the client
		for key, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	})

	// Start the proxy server
	fmt.Println("Starting proxy server on :8080")
	http.ListenAndServe(":8080", nil)
}

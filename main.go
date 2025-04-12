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
		fmt.Println("Error opening file: ", err)
		os.Exit(1)
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

	if len(os.Args) < 2 {
		fmt.Println("Please provide the route json file as an argument.")
		os.Exit(1)
	}

	routeFile := os.Args[1]
	routes, err := parseRoutes(routeFile)
	if err != nil {
		fmt.Println("Error parsing routes:", err)
		return
	}
	fmt.Println("Parsed routes:", routes)

	http.HandleFunc("/", getHandler(routes))

	// Start the proxy server
	fmt.Println("Starting proxy server on :8080")
	http.ListenAndServe(":8080", nil)
}

type RouteOutput struct {
	Path      string
	ServerUrl string
}

func getRouteOutput(routes []Route, path string) (RouteOutput, bool) {
	for _, route := range routes {
		if len(path) >= len(route.UrlStartsWith) && path[:len(route.UrlStartsWith)] == route.UrlStartsWith {
			path = path[len(route.UrlStartsWith):]
			return RouteOutput{Path: path, ServerUrl: route.RedirectHost}, true
		}
	}

	return RouteOutput{}, false
}

func getHandler(routes []Route) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Define the backend server URL
		// Strip the "/console" prefix from the URL path
		routeOutput, found := getRouteOutput(routes, r.URL.Path)
		path := r.URL.Path
		if found {
			fmt.Println("Forwarding request to backend server:", routeOutput.ServerUrl+routeOutput.Path)
			path = routeOutput.ServerUrl + routeOutput.Path
		}

		// Create a new request to the backend server

		req, err := http.NewRequest(r.Method, path, r.Body)
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
	}
}

package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

var (
	cache = make(map[string][]byte) // in-memory cache for cached responses
)

type ProxyInfo struct {
	URL string
}

func handleRequestAndRedirect(w http.ResponseWriter, r *http.Request) {
	// Check if the requested URL is in the whitelist
	/*allowedHosts := []string{"example.com", "google.com"} // replace with your list of allowed hosts
	if !isHostAllowed(r.Host, allowedHosts) {
		http.Error(w, "Access Denied", http.StatusForbidden)
		return
	}*/

	// Check if the response is in the cache and return it if found
	cacheKey := r.URL.String()
	if cachedResponse, ok := cache[cacheKey]; ok {
		fmt.Println("Serving response from cache for", cacheKey)
		w.Write(cachedResponse)
		return
	}

	// Modify the request to update the URL
	targetURL, _ := url.Parse("http://localhost:8000") // replace with your destination server URL
	//targetURL, _ := url.Parse("https://www.youtube.com/") // replace with your destination server URL
	//proxy := httputil.NewSingleHostReverseProxy(targetURL)
	r.URL.Host = targetURL.Host
	r.URL.Scheme = targetURL.Scheme
	r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
	r.Host = targetURL.Host

	// Forward the request to the destination server
	resp, err := http.DefaultTransport.RoundTrip(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	// Copy the response headers and body back to the client response
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)

	// Cache the response if it is cacheable
	if isCacheable(resp) {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		cache[cacheKey] = bodyBytes
		fmt.Println("Caching response for", cacheKey)
		resp.Body = ioutil.NopCloser(bytes.NewReader(bodyBytes))
	}
	defer resp.Body.Close()

	// Copy the response body back to the client response
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	w.Write(bodyBytes)
}

/*func isHostAllowed(host string, allowedHosts []string) bool {
	for _, h := range allowedHosts {
		if h == host {
			return true
		}
	}
	return false
}

// isBlacklisted checks if the given host is blacklisted
func isBlacklisted(host string) bool {
	blacklist := []string{"https://www.reddit.com", "example.org"} // replace with your blacklist
	for _, bl := range blacklist {
		if host == bl {
			return true
		}
	}
	return false
}*/

func isCacheable(resp *http.Response) bool {
	if resp.StatusCode != http.StatusOK {
		return false
	}
	if !strings.Contains(resp.Header.Get("Cache-Control"), "max-age=") {
		return false
	}
	if _, err := strconv.Atoi(strings.TrimPrefix(resp.Header.Get("Cache-Control"), "max-age=")); err != nil {
		return false
	}
	return true
}

func main() {
	port := ":7000" // replace with the port you want to listen on

	// Define the HTML template
	html := `
		<!DOCTYPE html>
		<html lang="en">
			<head>
				<meta charset="UTF-8">
				<meta name="viewport" content="width=device-width, initial-scale=1.0">
				<meta http-equiv="X-UA-Compatible" content="ie=edge">
				<title>Simple Proxy</title>
				<style>
					body {
						font-family: Arial, sans-serif;
						margin: 0;
						padding: 0;
						background-color: #000000;
						color: #ffffff;
					}
					h1 {
						font-size: 3rem;
						text-align: center;
						margin-top: 2rem;
					}
					form {
						display: flex;
						flex-direction: column;
						align-items: center;
						margin-top: 2rem;
					}
					label {
						font-size: 1.2rem;
					}
					input[type="text"] {
						padding: 0.5rem;
						font-size: 1.2rem;
						border-radius: 0.25rem;
						border: none;
						margin-top: 0.5rem;
						background-color: #ffffff;
						color: #000000;
					}
					button[type="submit"] {
						padding: 0.5rem 1rem;
						font-size: 1.2rem;
						background-color: #007bff;
						color: white;
						border: none;
						border-radius: 0.25rem;
						margin-top: 1rem;
						cursor: pointer;
					}
					h2 {
						font-size: 2rem;
						text-align: center;
						margin-top: 2rem;
					}
					img {
						display: block;
						margin: 2rem auto;
						max-width: 100%;
						height: auto;
					}
				</style>
			</head>
			<body>
				<h1>Simple Proxy</h1>
				<form method="GET" action="/">
					<label for="url">Enter the URL to proxy:</label>
					<input type="text" name="url" value="{{ .URL }}" placeholder="https://www.example.com">
					<button type="submit">Go</button>
				</form>
				{{ if ne .URL "" }}
					<h2>Proxying {{ .URL }}</h2>
				{{ end }}
				<img src="https://cdn.proprivacy.com/storage/images/proprivacy/2019/11/proxy-graphic-01-featured-image-defaultpng-content_image-default.png">
			</body>
		</html>
	`

	// Create a new HTTP server and set the request handler
	server := &http.Server{
		Addr: port,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Render the HTML template with the
			type templateData struct {
				URL string
			}
			tmpl, err := template.New("index").Parse(html)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			data := templateData{
				URL: "",
			}
			if r.Method == http.MethodGet {
				data.URL = r.URL.Query().Get("url")
				if data.URL != "" {
					// Check if the response is in the cache and return it if found
					cacheKey := r.URL.String()
					if cachedResponse, ok := cache[cacheKey]; ok {
						fmt.Println("Serving response from cache for", cacheKey)
						w.Write(cachedResponse)
						return
					}

					// Modify the request to update the URL
					targetURL, _ := url.Parse(data.URL)
					r.URL.Host = targetURL.Host
					r.URL.Scheme = targetURL.Scheme
					r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
					r.Host = targetURL.Host

					// Forward the request to the destination server
					resp, err := http.DefaultTransport.RoundTrip(r)
					if err != nil {
						http.Error(w, err.Error(), http.StatusBadGateway)
						return
					}

					// Copy the response headers and body back to the client response
					for k, vv := range resp.Header {
						for _, v := range vv {
							w.Header().Add(k, v)
						}
					}
					w.WriteHeader(resp.StatusCode)

					// Cache the response if it is cacheable
					if isCacheable(resp) {
						bodyBytes, err := ioutil.ReadAll(resp.Body)
						if err != nil {
							log.Fatal(err)
						}
						cache[cacheKey] = bodyBytes
						fmt.Println("Caching response for", cacheKey)
						resp.Body = ioutil.NopCloser(bytes.NewReader(bodyBytes))
					}
					defer resp.Body.Close()

					// Copy the response body back to the client response
					bodyBytes, err := ioutil.ReadAll(resp.Body)
					if err != nil {
						log.Fatal(err)
					}
					w.Write(bodyBytes)
				}
			}
			tmpl.Execute(w, data)
		}),
	}

	// Start the server and log any errors
	fmt.Printf("Starting server on port %s...\n", port)
	if err := server.ListenAndServe(); err != nil {
		fmt.Printf("Server error: %s\n", err.Error())
	}
}

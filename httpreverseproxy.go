package main

import (
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"time"

	"golang.org/x/net/proxy"
)

func StartHTTPReverseProxyServer(done chan int) {
	// Get the proxy server host and port from the PROXY_HOST environment variable.
	proxyHost := os.Getenv("PROXY_HOST")

	if proxyHost == "" {
		// If PROXY_HOST is not set, use the default proxy server.
		proxyHost = "127.0.0.1:1089"
	}

	// Create a Dialer with a timeout and keep-alive period of 30 seconds.
	baseDialer := net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	// Create a SOCKS5 dialer using the proxy server host and port, and the base dialer.
	dialSocksProxy, err := proxy.SOCKS5("tcp", proxyHost, nil, &baseDialer)
	if err != nil {
		// If there's an error creating the dialer, log it and return.
		log.Print(err)
		return
	}

	// Convert the SOCKS5 dialer to a ContextDialer.
	contextDialer := dialSocksProxy.(proxy.ContextDialer)

	// Define an HTTP handler function that sets up a reverse proxy for incoming requests.
	http.HandleFunc("/", func(w http.ResponseWriter, target *http.Request) {
		// Log the details of incoming requests.
		log.Printf("HTTP Reverse Proxy = host:%s, requrl:%s, method:%s\n", target.Host, target.RequestURI, target.Method)
		// Create a reverse proxy using the SOCKS5 dialer and the Director function to set the scheme and host of outgoing requests.
		reverseProxy := &httputil.ReverseProxy{
			Transport: &http.Transport{
				Proxy:                 http.ProxyFromEnvironment,
				DialContext:           contextDialer.DialContext,
				MaxIdleConns:          10,
				IdleConnTimeout:       60 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			},
			Director: func(req *http.Request) {
				req.URL.Scheme = "http"
				req.URL.Host = target.Host
			},
		}

		// Serve the incoming request using the reverse proxy.
		reverseProxy.ServeHTTP(w, target)
	})

	// Start the HTTP server to listen on port 80 and serve incoming requests.
	log.Println("Listening on port :80 ...")
	err = http.ListenAndServe(":80", nil)
	if err != nil {
		// If there's an error starting the server, panic.
		panic(err)
	}
}

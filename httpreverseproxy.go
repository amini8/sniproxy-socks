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
	proxyHost := os.Getenv("PROXY_HOST")

	if proxyHost == "" {
		proxyHost = "127.0.0.1:1089"
	}

	baseDialer := net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	dialSocksProxy, err := proxy.SOCKS5("tcp", proxyHost, nil, &baseDialer)
	if err != nil {
		log.Print(err)
		return
	}

	contextDialer := dialSocksProxy.(proxy.ContextDialer)

	http.HandleFunc("/", func(w http.ResponseWriter, target *http.Request) {
		log.Printf("HTTP Reverse Proxy = host:%s, requrl:%s, method:%s\n", target.Host, target.RequestURI, target.Method)
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

		reverseProxy.ServeHTTP(w, target)
	})
	log.Println("Listening on port :80 ...")
	err = http.ListenAndServe(":80", nil)
	if err != nil {
		panic(err)
	}
}

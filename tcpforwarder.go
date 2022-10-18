package main

import (
	"io"
	"log"
	"net"
	"os"
	"time"

	"golang.org/x/net/proxy"
)

func StartTCPForwarderServer(source, target string, done chan int) {
	listener, err := net.Listen("tcp", source)
	if err != nil {
		log.Fatalf("Failed to setup listener: %v", err)
	}

	log.Println("listening TCP forwarder on", source, " and forwarding to", target)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatalf("ERROR: failed to accept listener: %v", err)
		}
		log.Printf("Accepted connection from %v\n", conn.RemoteAddr().String())
		go forward(conn, target)
	}
}

func forward(sourceConn net.Conn, target string) {
	backendDialer := net.Dialer{
		Timeout: 5 * time.Second,
	}

	proxyHost := os.Getenv("PROXY_HOST")

	if proxyHost == "" {
		proxyHost = "127.0.0.1:1089"
	}

	dialSocksProxy, err := proxy.SOCKS5("tcp", proxyHost, nil, &backendDialer)
	if err != nil {
		log.Print(err)
		return
	}

	targetConn, err := dialSocksProxy.Dial("tcp", target)

	if err != nil {
		log.Print(err)
		return
	}

	if err != nil {
		log.Printf("Dial failed: %v", err)
		defer targetConn.Close()
		return
	}
	log.Printf("Forwarding from %v to %v\n", sourceConn.LocalAddr(), targetConn.RemoteAddr())
	go func() {
		defer targetConn.Close()
		defer sourceConn.Close()
		io.Copy(targetConn, sourceConn)
	}()
	go func() {
		defer targetConn.Close()
		defer sourceConn.Close()
		io.Copy(sourceConn, targetConn)
	}()
}

package main

import (
	"context"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"golang.org/x/net/proxy"
)

func StartTCPForwarderServer(ctx context.Context, source, target string) error {
	listener, err := net.Listen("tcp", source)
	if err != nil {
		return err
	}
	defer listener.Close()

	log.Printf("listening TCP forwarder on %s and forwarding to %s", source, target)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			conn, err := listener.Accept()
			if err != nil {
				log.Printf("failed to accept listener: %v", err)
				continue
			}
			log.Printf("accepted connection from %v", conn.RemoteAddr().String())
			go forward(ctx, conn, target)
		}
	}
}

func forward(ctx context.Context, sourceConn net.Conn, target string) {
	defer sourceConn.Close()

	backendDialer := net.Dialer{
		Timeout: 5 * time.Second,
	}

	proxyHost := os.Getenv("PROXY_HOST")
	if proxyHost == "" {
		proxyHost = "127.0.0.1:1089"
	}

	dialSocksProxy, err := proxy.SOCKS5("tcp", proxyHost, nil, &backendDialer)
	if err != nil {
		log.Printf("failed to dial SOCKS5 proxy: %v", err)
		return
	}

	targetConn, err := dialSocksProxy.Dial("tcp", target)
	if err != nil {
		log.Printf("failed to dial target: %v", err)
		return
	}
	defer targetConn.Close

package main

import (
	"bytes"
	"crypto/tls"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"golang.org/x/net/proxy"
)

func StartSNIProxyServer(done chan int) {
	// Listen for incoming connections on port 443
	l, err := net.Listen("tcp", ":443")
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Listening on port :443 ...")

	for {
		// Accept a new incoming connection
		conn, err := l.Accept()
		if err != nil {
			log.Print(err)
			continue
		}
		// Handle the connection in a new goroutine
		go handleConnection(conn)
	}

	// TODO graceful shutdown
}

// Peek at the client hello message to determine the server name
func peekClientHello(reader io.Reader) (*tls.ClientHelloInfo, io.Reader, error) {
	peekedBytes := new(bytes.Buffer)
	hello, err := readClientHello(io.TeeReader(reader, peekedBytes))
	if err != nil {
		return nil, nil, err
	}
	return hello, io.MultiReader(peekedBytes, reader), nil
}

// A connection that only allows reading
type readOnlyConn struct {
	reader io.Reader
}

func (conn readOnlyConn) Read(p []byte) (int, error)         { return conn.reader.Read(p) }
func (conn readOnlyConn) Write(p []byte) (int, error)        { return 0, io.ErrClosedPipe }
func (conn readOnlyConn) Close() error                       { return nil }
func (conn readOnlyConn) LocalAddr() net.Addr                { return nil }
func (conn readOnlyConn) RemoteAddr() net.Addr               { return nil }
func (conn readOnlyConn) SetDeadline(t time.Time) error      { return nil }
func (conn readOnlyConn) SetReadDeadline(t time.Time) error  { return nil }
func (conn readOnlyConn) SetWriteDeadline(t time.Time) error { return nil }

// Read the client hello message
func readClientHello(reader io.Reader) (*tls.ClientHelloInfo, error) {
	var hello *tls.ClientHelloInfo

	err := tls.Server(readOnlyConn{reader: reader}, &tls.Config{
		// GetConfigForClient is called to get the server config based on the client hello message
		GetConfigForClient: func(argHello *tls.ClientHelloInfo) (*tls.Config, error) {
			hello = new(tls.ClientHelloInfo)
			*hello = *argHello
			return nil, nil
		},
	}).Handshake()

	if hello == nil {
		return nil, err
	}

	return hello, nil
}

// Handle a single client connection
func handleConnection(clientConn net.Conn) {
	defer clientConn.Close()

	// Set a read deadline to ensure we don't block indefinitely on the client hello message
	if err := clientConn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		log.Print(err)
		return
	}

	// Peek at the client hello message to get the server name
	clientHello, clientReader, err := peekClientHello(clientConn)
	if err != nil {
		log.Print(err)
		return
	}

	// Clear the read deadline
	if err := clientConn.SetReadDeadline(time.Time{}); err != nil {
		log.Print(err)
		return
	}

	// Check if the server name is authorized
	// if !strings.HasSuffix(clientHello.ServerName, ".internal.example.com") {
	// 	log.Print("Blocking connection to unauthorized backend")
	// 	return
	// }

	// backendConn, err := net.DialTimeout("tcp", net.JoinHostPort(clientHello.ServerName, "443"), 5*time.Second)
backendDialer := net.Dialer{
	Timeout: 5 * time.Second,
}

// Check for the value of the PROXY_HOST environment variable, use default value if not set
	proxyHost := os.Getenv("PROXY_HOST")
	if proxyHost == "" {
		proxyHost = "127.0.0.1:1089"
	}

// Dial the SOCKS5 proxy with the given dialer and proxy server address
	dialSocksProxy, err := proxy.SOCKS5("tcp", proxyHost, nil, &backendDialer)
	if err != nil {
		log.Print(err)
		return
	}

// Log the SNI Proxy server name being accessed
	log.Printf("SNI Proxy = sni:%s", clientHello.ServerName)

// Dial the backend server using the SOCKS5 proxy
	backendConn, err := dialSocksProxy.Dial("tcp", net.JoinHostPort(clientHello.ServerName, "443"))
	if err != nil {
		log.Print(err)
		return
	}
	defer backendConn.Close()

	var wg sync.WaitGroup
	wg.Add(2)

// Copy data from the client connection to the backend connection
	go func() {
		io.Copy(clientConn, backendConn)
		clientConn.(*net.TCPConn).CloseWrite()
		wg.Done()
	}()

// Copy data from the backend connection to the client connection
	go func() {
		io.Copy(backendConn, clientReader)
		backendConn.(*net.TCPConn).CloseWrite()
		wg.Done()
	}()

	wg.Wait()
}

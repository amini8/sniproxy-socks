# sniproxy-socks
This is a Go program that acts as a proxy server and intercepts TLS connections, allowing traffic to be routed to a specified proxy server instead of the originally intended backend. Here's a brief explanation of the code:

The StartSNIProxyServer function sets up a TCP listener on port 443 and accepts incoming connections. For each connection, it spins off a new goroutine to handle the connection by calling handleConnection.

The peekClientHello function reads the first bytes of a TLS connection to determine the server name indicated in the client hello message. It returns the tls.ClientHelloInfo struct containing the server name and a new reader that can be used to read the rest of the connection data.

The readOnlyConn struct is a read-only net.Conn implementation that can be used to pass the client hello data to tls.Server for parsing.

The readClientHello function uses the tls.Server function to parse the client hello message and extract the server name. It returns the tls.ClientHelloInfo struct containing the server name.

The handleConnection function first reads the client hello message using peekClientHello and extracts the server name using readClientHello. It then sets up a connection to the specified proxy server using the SOCKS5 protocol, and uses goroutines to copy data between the client and the backend.

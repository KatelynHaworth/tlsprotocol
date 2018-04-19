package tlsprotocol

import (
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"strconv"
	"syscall"
)

// Listener is a TLS connection listener
// that supports the use of multiple sockets
// for receiving connections and also supports
// breaking ALPN protocols into specific listeners
type Listener struct {
	// BindAddr specifies the hostname or IP address
	// and port to bind listening sockets too
	BindAddr string

	// TLSConfig is the TLS configuration used to
	// build the TLS listener sockets, ensure that
	// all required protocols are configured otherwise
	// the listener will refuse to listen specifically for
	// the Protocol and will direct it to the default queue
	TLSConfig *tls.Config

	// Listeners specifies the number of underlying
	// sockets to bind for receiving connections, if
	// not set it will default to 1
	Listeners int

	// workers stores the references to the underlying
	// listen workers that listen for connections from
	// their socket
	workers []*worker

	// addr is the parsed BindAddr as a
	// net.Addr struct
	addr net.Addr

	// sockAddr is the parsed BindAddr as
	// a socket address
	sockAddr syscall.Sockaddr

	// channels is a map of ALPN Protocol
	// names to their Protocol channels
	channels map[string]*Protocol

	// defaultChannel is the channel that receives
	// connections that don't match any of the explicitly
	// declared protocols
	defaultChannel chan net.Conn

	// errors receives errors from listen workers
	// and is piped out via the default Accept() handle
	errors chan error
}

// Start initialises the TLS listener by spawning
// workers to receive connections and constructs the
// channels to receive default connections and errors
func (listener *Listener) Start() error {
	if listener.Listeners == 0 {
		listener.Listeners = 1
	}

	listener.workers = make([]*worker, listener.Listeners)
	listener.defaultChannel = make(chan net.Conn, 1)
	listener.errors = make(chan error, 1)

	for i := range listener.workers {
		socket, err := listener.buildSocket()
		if err != nil {
			listener.Stop()
			return fmt.Errorf("builder worker socket: %s", err)
		}

		listener.workers[i] = &worker{
			parent: listener,
			socket: socket,
		}

		listener.workers[i].start()
	}

	return nil
}

// Accept will receive connections from the
// default channel (i.e. connections that didn't
// match an accepted Protocol), it also receives
// all worker errors
func (listener *Listener) Accept() (net.Conn, error) {
	select {
	case conn, ok := <-listener.defaultChannel:
		if !ok {
			return nil, fmt.Errorf("accept %s %s: use of closed network connection", listener.addr.Network(), listener.addr.String())
		}

		return conn, nil

	case err := <-listener.errors:
		return nil, err
	}
}

// Protocol setups a net.Listener to receive all
// TLS connections that match the ALPN Protocol
func (listener *Listener) Protocol(proto string) (net.Listener, error) {
	if len(listener.workers) > 0 {
		return nil, fmt.Errorf("protocol listener must be created before starting listener")
	}

	if _, exists := listener.channels[proto]; exists {
		return nil, fmt.Errorf("protocol listener already declared for proto: %s", proto)
	}

	if !listener.protocolConfigured(proto) {
		return nil, fmt.Errorf("protocol not specified in the TLS configuration: %s", proto)
	}

	if listener.channels == nil {
		listener.channels = make(map[string]*Protocol, 0)
	}

	listener.channels[proto] = &Protocol{
		parent:  listener,
		proto:   proto,
		channel: make(chan net.Conn, 1),
	}

	return listener.channels[proto], nil
}

// Addr returns the address that the
// listener will receive connections on
func (listener *Listener) Addr() net.Addr {
	return listener.addr
}

// Close calls the Stop() functions on
// the listener
func (listener *Listener) Close() error {
	listener.Stop()
	return nil
}

// Stop will stop all the workers before
// closing Protocol listener channels and
// finally closes the default channel
func (listener *Listener) Stop() {
	for i := range listener.workers {
		if listener.workers[i] != nil {
			listener.workers[i].stop()
		}
	}

	for proto := range listener.channels {
		listener.channels[proto].Close()
	}

	if len(listener.defaultChannel) == 1 {
		conn := <-listener.defaultChannel
		conn.Close()
	}

	close(listener.defaultChannel)
	listener.workers = nil
	listener.channels = nil
	listener.sockAddr = nil
}

// protocolConfigured checks if the provided ALPN Protocol
// has been specified in the `NextProtos` sections of the
// TLS configuration
func (listener *Listener) protocolConfigured(proto string) bool {
	for i := range listener.TLSConfig.NextProtos {
		if listener.TLSConfig.NextProtos[i] == proto {
			return true
		}
	}

	return false
}

// connectionReceived is called by works to send
// connections up to the parent listener for the
// connection to be sorted into a channel based on
// the negotiated ALPN Protocol
func (listener *Listener) connectionReceived(conn net.Conn) {
	tlsConn := conn.(*tls.Conn)
	if err := tlsConn.Handshake(); err != nil {
		tlsConn.Close()
		return
	}

	if proto, ok := listener.channels[tlsConn.ConnectionState().NegotiatedProtocol]; ok && tlsConn.ConnectionState().NegotiatedProtocolIsMutual {
		proto.channel <- tlsConn
	} else {
		listener.defaultChannel <- tlsConn
	}
}

// getSocketAddress will parse the `BindAddr` into
// a socket address that a socket can be bound to,
// `BindAddr` is only parsed once and then stored in
// the listener struct to prevent excess operations
func (listener *Listener) getSocketAddress() (syscall.Sockaddr, error) {
	if listener.sockAddr != nil {
		return listener.sockAddr, nil
	}

	host, port, err := net.SplitHostPort(listener.BindAddr)
	if err != nil {
		return nil, fmt.Errorf("split listener address to host and port: %s", err)
	}

	portInt, err := strconv.ParseInt(port, 10, 16)
	if err != nil {
		return nil, fmt.Errorf("parse listener address port to int: %s", err)
	}

	addr, err := net.ResolveIPAddr("ip", host)
	if err != nil {
		return nil, fmt.Errorf("resolove listener address: %s", err)
	}

	switch len(addr.IP) {
	case net.IPv4len:
		ip := [4]byte{}
		copy(ip[:], addr.IP)
		listener.sockAddr = &syscall.SockaddrInet4{Addr: ip, Port: int(portInt)}

	case net.IPv6len:
		ip := [16]byte{}
		copy(ip[:], addr.IP)
		listener.sockAddr = &syscall.SockaddrInet6{Addr: ip, Port: int(portInt)}

	default:
		return nil, fmt.Errorf("invalid IP address length: %d", len(addr.IP))
	}

	listener.addr = &net.TCPAddr{IP: addr.IP, Zone: addr.Zone, Port: int(portInt)}
	return listener.sockAddr, nil
}

// buildSocket opens a socket in the kernel,
// sets the socket options to allow multiple binds,
// binds the socket and finally starts it listening
func (listener *Listener) buildSocket() (net.Listener, error) {
	socketAddress, err := listener.getSocketAddress()
	if err != nil {
		return nil, fmt.Errorf("get socket address for bind: %s", err)
	}

	inetFamily := syscall.AF_INET
	if _, ok := socketAddress.(*syscall.SockaddrInet6); ok {
		inetFamily = syscall.AF_INET6
	}

	fileDescriptor, err := syscall.Socket(inetFamily, syscall.SOCK_STREAM, syscall.IPPROTO_TCP)
	if err != nil {
		return nil, fmt.Errorf("unable to create socket in kernel: %s", err)
	}

	if err = syscall.SetsockoptInt(fileDescriptor, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
		syscall.Close(fileDescriptor)
		return nil, fmt.Errorf("failed to set SO_REUSEADDR on socket: %s", err)
	}

	if err = syscall.SetsockoptInt(fileDescriptor, syscall.SOL_SOCKET, so_reuseport, 1); err != nil {
		syscall.Close(fileDescriptor)
		return nil, fmt.Errorf("failed to set SO_REUSEPORT on socket: %s", err)
	}

	if err = syscall.SetNonblock(fileDescriptor, true); err != nil {
		syscall.Close(fileDescriptor)
		return nil, fmt.Errorf("failed to set non-blocking on socket: %s", err)
	}

	if err = syscall.Bind(fileDescriptor, socketAddress); err != nil {
		syscall.Close(fileDescriptor)
		return nil, fmt.Errorf("failed to bind socket to address: %s", err)
	}

	if err = syscall.Listen(fileDescriptor, syscall.SOMAXCONN); err != nil {
		syscall.Close(fileDescriptor)
		return nil, fmt.Errorf("failed to start listening for socket: %s", err)
	}

	socket, err := net.FileListener(os.NewFile(uintptr(fileDescriptor), "tls-Protocol-listener"))
	if err != nil {
		syscall.Close(fileDescriptor)
		return nil, fmt.Errorf("failed to convert file descriptor to listener: %s", err)
	}

	return tls.NewListener(socket, listener.TLSConfig), nil
}

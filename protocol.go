package tlsprotocol

import (
	"fmt"
	"net"
)

// Protocol is a `net.Listener` interface
// that receives connections from the parent
// listener for the specific ALPN Protocol
// configured
type Protocol struct {
	parent  *Listener
	proto   string
	channel chan net.Conn
}

// Accept will block until a new connection
// is available in the Protocol's channel
func (protocol *Protocol) Accept() (net.Conn, error) {
	if conn, open := <-protocol.channel; !open {
		return nil, fmt.Errorf("use of closed socket")
	} else {
		return conn, nil
	}
}

// Close will close the Protocol's channel
// so it can't receive any more connections
// and will remove itself from the parent Listener.
//
// If the Protocol is closed but not the parent all
// connections for it's ALPN Protocol will be directed
// to the default channel.
func (protocol *Protocol) Close() error {
	if _, ok := protocol.parent.channels[protocol.proto]; ok {
		close(protocol.channel)
		delete(protocol.parent.channels, protocol.proto)
		return nil
	}

	return fmt.Errorf("listener already closed")
}

// Addr returns the address the parent listener
// is receiving connections on
func (protocol *Protocol) Addr() net.Addr {
	return protocol.parent.addr
}

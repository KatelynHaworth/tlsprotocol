package tlsprotocol

import (
	"net"
)

// worker is a standalone socket that
// listens for connections and then sends
// those connections back to the parent
// listener for handling
type worker struct {
	parent  *Listener
	running bool
	socket  net.Listener
}

// start sets the internal state of
// the worker to running and spawns
// a go routine for receiving connections
// from the configured socket
func (worker *worker) start() {
	if worker.running {
		return
	}

	worker.running = true
	go worker.listen()
}

// listen will receive connections from
// the configured socket for the worker
// until the internal state of the worker
// is changed to no running
func (worker *worker) listen() {
	for worker.running {
		conn, err := worker.socket.Accept()
		if err != nil {
			worker.parent.errors <- err
			continue
		}

		go worker.parent.connectionReceived(conn)
	}
}

// stop sets the internal state of
// the worker to not running and closes
// the configured socket
func (worker *worker) stop() {
	worker.running = false
	worker.socket.Close()
}

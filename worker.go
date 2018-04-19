package tlsprotocol

import (
	"net"
	"sync"
)

// worker is a standalone socket that
// listens for connections and then sends
// those connections back to the parent
// listener for handling
type worker struct {
	parent  *Listener
	running bool
	socket  net.Listener
	lock    sync.Mutex
}

// start sets the internal state of
// the worker to running and spawns
// a go routine for receiving connections
// from the configured socket
func (worker *worker) start() {
	if worker.isRunning() {
		return
	}

	worker.lock.Lock()
	defer worker.lock.Unlock()
	worker.running = true

	go worker.listen()
}

// isRunning will return the value of
// `running` of the worker but in a race
// safe way
func (worker *worker) isRunning() bool {
	worker.lock.Lock()
	defer worker.lock.Unlock()
	return worker.running
}

// listen will receive connections from
// the configured socket for the worker
// until the internal state of the worker
// is changed to no running
func (worker *worker) listen() {
	for worker.isRunning() {
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
	worker.lock.Lock()
	defer worker.lock.Unlock()

	worker.running = false
	worker.socket.Close()
}

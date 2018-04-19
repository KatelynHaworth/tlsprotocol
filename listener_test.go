package tlsprotocol

import (
	"crypto/tls"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net"
)

var _ = Describe("Listener", func() {
	var h2Listener net.Listener

	cert, _ := tls.LoadX509KeyPair("test_certificate.crt", "test_certificate.key")
	listener := &Listener{
		Listeners: 0,
		BindAddr:  "127.0.0.1:6080",
		TLSConfig: &tls.Config{
			NextProtos:   []string{"h2"},
			Certificates: []tls.Certificate{cert},
		},
	}

	It("Shouldn't allow a protocol listener for a proto not in the TLS config", func() {
		protoListener, err := listener.Protocol("h3")

		Expect(protoListener).To(BeNil())
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(Equal("protocol not specified in the TLS configuration: h3"))
	})

	It("Should configure a protocol listener for a configured proto", func() {
		protoListener, err := listener.Protocol("h2")

		Expect(protoListener).ToNot(BeNil())
		Expect(err).To(BeNil())

		Expect(protoListener).To(BeAssignableToTypeOf(&Protocol{}))
		Expect(protoListener.(*Protocol).parent).To(Equal(listener))
		Expect(protoListener.(*Protocol).proto).To(Equal("h2"))

		h2Listener = protoListener
	})

	It("Shouldn't allow a protocol listener to be configured more than once", func() {
		protoListener, err := listener.Protocol("h2")

		Expect(protoListener).To(BeNil())
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(Equal("protocol listener already declared for proto: h2"))
	})

	It("Should bind listening sockets", func() {
		err := listener.Start()
		Expect(err).Should(BeNil())

		Expect(listener.defaultChannel).ToNot(BeNil())
		Expect(listener.errors).ToNot(BeNil())
		Expect(listener.addr).ToNot(BeNil())
		Expect(listener.sockAddr).ToNot(BeNil())
		Expect(len(listener.workers)).To(Equal(1))

		Expect(listener.Addr()).To(BeAssignableToTypeOf(&net.TCPAddr{}))
		Expect(listener.Addr().(*net.TCPAddr).Port).To(Equal(6080))
		Expect(listener.Addr().(*net.TCPAddr).IP).To(Equal(net.IP{
			0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xFF, 0xFF, 0x7f, 0x0, 0x0, 0x01, // 127.0.0.1 as net.IPv6
		}))

		for _, worker := range listener.workers {
			Expect(worker.running).To(Equal(true))
			Expect(worker.parent).To(Equal(listener))
			Expect(worker.socket).ToNot(BeNil())

			Expect(worker.socket.Addr()).To(BeAssignableToTypeOf(&net.TCPAddr{}))
			Expect(worker.socket.Addr().(*net.TCPAddr).Port).To(Equal(6080))
			Expect(worker.socket.Addr().(*net.TCPAddr).IP).To(Equal(net.IP{
				0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xFF, 0xFF, 0x7f, 0x0, 0x0, 0x01, // 127.0.0.1 as net.IPv6
			}))
		}

		for _, channel := range listener.channels {
			Expect(channel.Addr()).To(BeAssignableToTypeOf(&net.TCPAddr{}))
			Expect(channel.Addr().(*net.TCPAddr).Port).To(Equal(6080))
			Expect(channel.Addr().(*net.TCPAddr).IP).To(Equal(net.IP{
				0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xFF, 0xFF, 0x7f, 0x0, 0x0, 0x01, // 127.0.0.1 as net.IPv6
			}))
		}
	})

	It("Shouldn't allow a protocol listen to be configured after starting", func() {
		protoListener, err := listener.Protocol("h3")

		Expect(protoListener).To(BeNil())
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(Equal("protocol listener must be created before starting listener"))
	})

	It("Should accept TLS connections and queue them to the correct channel", func() {
		Expect(len(listener.defaultChannel)).To(Equal(0))
		Expect(len(listener.channels["h2"].channel)).To(Equal(0))

		conn, err := tls.Dial("tcp", "127.0.0.1:6080", &tls.Config{InsecureSkipVerify: true})

		Expect(err).To(BeNil())
		Expect(conn).ToNot(BeNil())
		Expect(conn.Handshake()).To(BeNil())
		Expect(len(listener.defaultChannel)).To(Equal(1))

		conn, err = tls.Dial("tcp", "127.0.0.1:6080", &tls.Config{InsecureSkipVerify: true, NextProtos: []string{"h2"}})

		Expect(err).To(BeNil())
		Expect(conn).ToNot(BeNil())
		Expect(conn.Handshake()).To(BeNil())
		Expect(len(listener.channels["h2"].channel)).To(Equal(1))
	})

	It("Should return connections queued in the default channel", func() {
		conn, err := listener.Accept()
		defer conn.Close()

		Expect(err).To(BeNil())
		Expect(conn).ToNot(BeNil())
		Expect(len(listener.defaultChannel)).To(Equal(0))
	})

	It("Should return connections queued in a protocols channel", func() {
		conn, err := h2Listener.Accept()
		defer conn.Close()

		Expect(err).To(BeNil())
		Expect(conn).ToNot(BeNil())
		Expect(len(listener.channels["h2"].channel)).To(Equal(0))
	})

	It("Should stop listening sockets and cleanup", func() {
		listener.Stop()
		Expect(listener.defaultChannel).To(BeClosed())
		Expect(listener.sockAddr).To(BeNil())
		Expect(len(listener.workers)).To(Equal(0))
	})

	It("Should return an error when accepting on a closed listener", func() {
		conn, err := listener.Accept()

		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(Equal(fmt.Sprintf("accept %s %s: use of closed network connection", listener.Addr().Network(), listener.Addr().String())))
		Expect(conn).To(BeNil())

		conn, err = h2Listener.Accept()

		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(Equal(fmt.Sprintf("accept %s %s: use of closed network connection", h2Listener.Addr().Network(), h2Listener.Addr().String())))
		Expect(conn).To(BeNil())
	})
})

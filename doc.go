// Package tlsprotocol provides an abstraction on top of the TLS listener functionality to provide the ability to have
// individual net.Listeners for application layer protocols (ALPN) negotiated during the TLS handshake between the client
// and server.
//
// For example, lets say you have a server that you want to be able to accept generic TLS connections but split HTTP/2
// into a separate listener for handling in a different way.
//
//   package main
//
//   import (
//   	"crypto/tls"
//   	"fmt"
//   	"log"
//
//   	"github.com/LiamHaworth/tlsprotocol"
//   )
//
//   func main() {
//   	log.Print("Loading key pair cert.pem/key.pem")
//   	certs, err := tls.LoadX509KeyPair("cert.pem", "key.pem")
//   	if err != nil {
//   		log.Fatal(err)
//   	}
//
//   	log.Print("Configuring protocol listener")
//   	srv := &tlsprotocol.Listener{
//   		BindAddr:  "127.0.0.1:443",
//   		Listeners: 1,
//   		TLSConfig: &tls.Config{
//   			Certificates: []tls.Certificate{certs},
//   			NextProtos:   []string{"h2"},
//   		},
//   	}
//
//   	log.Print("Setting up listener for HTTP\\2")
//   	h2Listener, err := srv.Protocol("h2")
//   	if err != nil {
//   		log.Fatal(err)
//   	}
//
//   	log.Print("Starting protocol listener")
//   	if err := srv.Start(); err != nil {
//   		log.Fatal(err)
//   	}
//
//   	log.Print("Listening for connections")
//   	go func() {
//   		for {
//   			conn, _ := h2Listener.Accept()
//   			fmt.Println("Connection received via h2 (HTTP\\2)", conn)
//   		}
//   	}()
//
//   	for {
//   		conn, _ := srv.Accept()
//   		fmt.Println("Connection received via default", conn)
//   	}
//   }
package tlsprotocol

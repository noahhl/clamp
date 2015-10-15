package clamp

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"
)

const readLen = 256 * 1024
const channelBufferSize = 2000000

type Server struct {
	MessageChannel chan string
	name           string
	addr           string
	numProcessed   int
	numDropped     int
	Explode        bool
	Delim          string
}

func NewServer(name string, addr string) *Server {
	ch := make(chan string, channelBufferSize)
	s := Server{ch, name, addr, 0, 0, false, "\n"}
	go func() {
		c := time.Tick(1 * time.Second)
		for {
			<-c
			StatsChannel <- Stat{s.name + "ServerProcessed", fmt.Sprintf("%v", s.numProcessed)}
			StatsChannel <- Stat{s.name + "ServerDropped", fmt.Sprintf("%v", s.numDropped)}
			StatsChannel <- Stat{s.name + "MessageChannelSize", fmt.Sprintf("%v", len(s.MessageChannel))}
		}
	}()

	return &s
}

func (s *Server) processBytes(buf []byte) {
	pieces := strings.Split(string(buf), s.Delim)
	for i := range pieces {
		if len(pieces[i]) > 0 {
			select {
			case s.MessageChannel <- pieces[i]:
				s.numProcessed += 1
			default:
				s.numDropped += 1
				if s.Explode {
					fmt.Printf("%v: dropped a message on the %v input server, channel couldn't keep up\n", time.Now(), s.name)
					os.Exit(2)
				} else {
					fmt.Printf("%v: dropped a message on the %v input server, channel couldn't keep up\n", time.Now(), s.name)
				}
			}
		}
	}
}

func (s *Server) listenUDP() {
	go func() {
		listener, err := net.ListenPacket("udp", s.addr)

		if err != nil {
			panic(err)
		}

		defer listener.Close()
		buffer := make([]byte, readLen)
		for {
			n, _, err := listener.ReadFrom(buffer)
			if err != nil {
				continue
			}
			s.processBytes(buffer[0:n])
		}
	}()

}

func (s *Server) listenTCP() {
	go func() {

		addr, err := net.ResolveTCPAddr("tcp", s.addr)
		if err != nil {
			panic(err)
		}
		listener, err := net.ListenTCP("tcp", addr)
		defer listener.Close()
		if err != nil {
			panic(err)
		}
		conns := s.clientTCPConns(listener)
		for {
			go func(client *net.TCPConn) {
				b := bufio.NewReaderSize(client, readLen)
				for {
					buffer := make([]byte, 0)
					for len(buffer) < readLen && (len(buffer) < len(s.Delim) || string(buffer[(len(buffer)-len(s.Delim)):len(buffer)]) != s.Delim) {
						tmp_buf := make([]byte, readLen)
						n, err := b.Read(tmp_buf)
						if err != nil && err != io.EOF {
							return
						} else if err == io.EOF {
							break
						}
						buffer = append(buffer, tmp_buf[0:n]...)
					}
					s.processBytes(buffer)
				}
			}(<-conns)
		}
	}()

}

func (s *Server) clientTCPConns(listener *net.TCPListener) chan *net.TCPConn {
	ch := make(chan *net.TCPConn)
	go func() {
		for {
			client, err := listener.AcceptTCP()
			if client == nil {
				fmt.Printf("couldn't accept: %v", err)
				continue
			}
			ch <- client
		}
	}()
	return ch
}

func StartUDPServer(address string) chan string {
	server := NewServer("udp", address)
	server.listenUDP()
	return server.MessageChannel
}

func StartTCPServer(address string) chan string {
	server := NewServer("tcp", address)
	server.listenTCP()
	return server.MessageChannel
}

func StartDualServer(address string) chan string {
	server := NewServer("dual", address)
	server.listenUDP()
	server.listenTCP()
	return server.MessageChannel
}

func NewDualServer(address string) *Server {
	server := NewServer("dual", address)
	server.listenUDP()
	server.listenTCP()
	return server
}

func StartExplodingDualServer(address string) chan string {
	server := NewServer("dual", address)
	server.listenUDP()
	server.listenTCP()
	server.Explode = true
	return server.MessageChannel
}

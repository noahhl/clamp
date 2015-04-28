package clamp

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"
)

const readLen = 16392
const channelBufferSize = 2000000

type Server struct {
	messageChannel chan string
	name           string
	addr           string
	numProcessed   int
	numDropped     int
	Explode bool
}

func NewServer(name string, addr string) *Server {
	ch := make(chan string, channelBufferSize)
	s := Server{ch, name, addr, 0, 0, false}
	go func() {
		c := time.Tick(1 * time.Second)
		for {
			<-c
			StatsChannel <- Stat{s.name + "ServerProcessed", fmt.Sprintf("%v", s.numProcessed)}
			StatsChannel <- Stat{s.name + "ServerDropped", fmt.Sprintf("%v", s.numDropped)}
			StatsChannel <- Stat{s.name + "MessageChannelSize", fmt.Sprintf("%v", len(s.messageChannel))}
		}
	}()

	return &s
}

func (s *Server) processBytes(buf []byte) {
	pieces := strings.Split(string(buf), "\n")
	for i := range pieces {
		if len(pieces[i]) > 0 {
			select {
			case s.messageChannel <- pieces[i]:
				s.numProcessed += 1
			default:
				s.numDropped += 1
				if s.Explode {
				  panic(fmt.Sprintf("%v: dropped a message on the %v input server, channel couldn't keep up\n", time.Now(), s.name))
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

		listener, err := net.Listen("tcp", s.addr)
		defer listener.Close()
		if err != nil {
			panic(err)
		}
		conns := s.clientTCPConns(listener)
		for {
			go func(client net.Conn) {
				b := bufio.NewReader(client)
				for {
					line, err := b.ReadBytes('\n')
					if err != nil {
						return
					}
					s.processBytes(line)
				}
			}(<-conns)
		}
	}()

}

func (s *Server) clientTCPConns(listener net.Listener) chan net.Conn {
	ch := make(chan net.Conn)
	go func() {
		for {
			client, err := listener.Accept()
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
	return server.messageChannel
}

func StartTCPServer(address string) chan string {
	server := NewServer("tcp", address)
	server.listenTCP()
	return server.messageChannel
}

func StartDualServer(address string) chan string {
	server := NewServer("dual", address)
	server.listenUDP()
	server.listenTCP()
	return server.messageChannel
}

func StartExplodingDualServer(address string) chan string {
	server := NewServer("dual", address)
	server.listenUDP()
	server.listenTCP()
	server.Explode = true 
	return server.messageChannel
}

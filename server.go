package clamp

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"
)

const readLen = 8196
const channelBufferSize = 50000

func StartDualServer(address string) chan string {

	udp_server := StartUDPServer(address)
	tcp_server := StartTCPServer(address)
	messageChannel := make(chan string, channelBufferSize)

	go func() {
		for {
			select {
			case msg := <-udp_server:
				messageChannel <- msg
			case msg := <-tcp_server:
				messageChannel <- msg
			}
		}
	}()
	return messageChannel

}

func StartUDPServer(address string) chan string {
	messageChannel := make(chan string, channelBufferSize)

	numProcessed := 0
	numDropped := 0
	go func() {
		c := time.Tick(5 * time.Second)
		for {
			<-c
			StatsChannel <- Stat{"udpServerProcessed", fmt.Sprintf("%v", numProcessed)}
			StatsChannel <- Stat{"udpServerDropped", fmt.Sprintf("%v", numDropped)}
		}
	}()

	go func() {
		server, err := net.ListenPacket("udp", address)

		if err != nil {
			panic(err)
		}

		defer server.Close()
		buffer := make([]byte, readLen)
		for {
			n, _, err := server.ReadFrom(buffer)
			if err != nil {
				continue
			}
			select {
			case messageChannel <- strings.TrimSpace(strings.Replace(string(buffer[0:n]), "\n", "", -1)):
				numProcessed += 1
			default:
				numDropped += 1
				fmt.Printf("%v: dropped a message on the UDP input server, channel couldn't keep up\n", time.Now())
			}
		}
	}()
	return messageChannel
}

func StartTCPServer(address string) chan string {
	messageChannel := make(chan string, channelBufferSize)

	numProcessed := 0
	numDropped := 0
	go func() {
		c := time.Tick(5 * time.Second)
		for {
			<-c
			StatsChannel <- Stat{"tcpServerProcessed", fmt.Sprintf("%v", numProcessed)}
			StatsChannel <- Stat{"tcpServerDropped", fmt.Sprintf("%v", numDropped)}
		}
	}()

	go func() {

		server, err := net.Listen("tcp", address)
		if err != nil {
			panic(err)
		}
		conns := clientTCPConns(server)
		for {
			go func(client net.Conn) {
				b := bufio.NewReader(client)
				for {
					line, err := b.ReadBytes('\n')
					if err != nil {
						return
					}
					select {
					case messageChannel <- strings.TrimSpace(strings.Replace(string(line), "\n", "", -1)):
						numProcessed += 1
					default:
						fmt.Printf("%v: dropped a message on the TCP input server, channel couldn't keep up\n", time.Now())
						numDropped += 1
					}
				}
			}(<-conns)
		}
	}()
	return messageChannel
}

func clientTCPConns(listener net.Listener) chan net.Conn {
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

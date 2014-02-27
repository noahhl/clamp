package clamp

import (
	"net"
	"testing"
	"time"
)

func TestUDPServer(t *testing.T) {
	//This is unfortunate -- ideally we'd let the OS pick a free port,
	//but need to rejigger server to expose the port that it started up on
	messageChannel := StartUDPServer("127.0.0.1:8125")
	time.Sleep(500 * time.Millisecond)
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:8125")
	if err != nil {
		panic(err)
	}
	client, err := net.ListenPacket("udp", ":0") //net.Dial("udp", ":8125")
	if err != nil {
		panic(err)
	}
	defer client.Close()
	client.(*net.UDPConn).WriteToUDP([]byte("test\n"), addr)

	time.Sleep(10 * time.Millisecond)
	if len(messageChannel) != 1 {
		t.Errorf("expected one message in channel, got %v\n", len(messageChannel))
	}

	message := <-messageChannel
	if message != "test" {
		t.Errorf("expected message to be 'test', was '%v'\n", message)
	}

	client.(*net.UDPConn).WriteToUDP([]byte("test\nbar"), addr)
	time.Sleep(10 * time.Millisecond)
	if len(messageChannel) != 2 {
		t.Fatalf("expected two messages in channel, got %v\n", len(messageChannel))
	}

	message = <-messageChannel
	if message != "test" {
		t.Errorf("expected message to be 'test', was '%v'\n", message)
	}

	message = <-messageChannel
	if message != "bar" {
		t.Errorf("expected message to be 'bar', was '%v'\n", message)
	}

}

func TestTCPServer(t *testing.T) {
	messageChannel := StartTCPServer("127.0.0.1:8125")
	time.Sleep(500 * time.Millisecond)
	client, err := net.Dial("tcp", "127.0.0.1:8125")
	if err != nil {
		panic(err)
	}
	defer client.Close()
	client.Write([]byte("test\n"))

	time.Sleep(10 * time.Millisecond)
	if len(messageChannel) != 1 {
		t.Errorf("expected one message in channel, got %v\n", len(messageChannel))
	}

	message := <-messageChannel
	if message != "test" {
		t.Errorf("expected message to be 'test', was '%v'\n", message)
	}
}

func TestDualServer(t *testing.T) {
	time.Sleep(500 * time.Millisecond)
	messageChannel := StartDualServer("127.0.0.1:8126")
	time.Sleep(500 * time.Millisecond)
	tcpClient, err := net.Dial("tcp", "127.0.0.1:8126")
	if err != nil {
		panic(err)
	}
	defer tcpClient.Close()
	tcpClient.Write([]byte("tcp_test\n"))

	time.Sleep(10 * time.Millisecond)
	if len(messageChannel) != 1 {
		t.Errorf("expected one message in channel, got %v\n", len(messageChannel))
	}

	message := <-messageChannel
	if message != "tcp_test" {
		t.Errorf("expected message to be 'tcp_test', was '%v'\n", message)
	}

	udpClient, err := net.Dial("udp", "127.0.0.1:8126")
	if err != nil {
		panic(err)
	}
	defer udpClient.Close()
	udpClient.Write([]byte("udp_test\n"))

	time.Sleep(10 * time.Millisecond)
	if len(messageChannel) != 1 {
		t.Errorf("expected one message in channel, got %v\n", len(messageChannel))
	}

	message = <-messageChannel
	if message != "udp_test" {
		t.Errorf("expected message to be 'udp_test', was '%v'\n", message)
	}

}

func BenchmarkMessageParsing(b *testing.B) {
	s := NewServer("test", ":8125")
	go func() {
		for {
			select {
			case _ = <-s.messageChannel:
			default:
			}
		}
	}()
	for j := 0; j < b.N; j++ {
		s.processBytes([]byte("foo\nbar\nbazalskdfjalskdfjalsdfjka\n"))
	}
}

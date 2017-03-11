package omsg

import (
	"log"
	"net"
	"testing"
)

var s *Server
var c *Client
var ch chan bool

const testData = "testData"

// Test ...
func Test(t *testing.T) {
	main()
}

func main() {
	ch = make(chan bool)
	s = NewServer(onServerData, onNewClient, onServerClose)
	s.StartServer("0.0.0.0:1234")

	c = NewClient(onClientData, onClientClose)
	if err := c.Connect("0.0.0.0:1234", true, 5, 5); err != nil {
		log.Fatalln("[C] connect error:", err)
	}
	if 0 == c.Send([]byte("c"+testData)) {
		log.Fatalln("[C] send error:0")
	}

	<-ch
	// time.Sleep(time.Second * 10)
}

func onServerData(conn net.Conn, data []byte) {
	if string(data) != "c"+testData {
		log.Fatalln("[S] recv data:", conn.RemoteAddr(), " => ", string(data))
	}
	s.SendToAll([]byte("s" + testData))
	conn.Close()
}

func onNewClient(conn net.Conn) {
	log.Println("[S] new client:", conn.RemoteAddr(), " new client.")
}

func onServerClose(conn net.Conn) {
	log.Println("[S]", conn.RemoteAddr(), " closed.")
}

func onClientData(data []byte) {
	if string(data) != "s"+testData {
		log.Fatalln("[C] client recv => ", string(data))
	}
	c.Close()
	ch <- true
}

func onClientClose() {
	log.Println("[C] closed!")
}

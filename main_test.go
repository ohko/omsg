package omsg

import (
	"bytes"
	crand "crypto/rand"
	"encoding/binary"
	"log"
	"net"
	"runtime"
	"testing"
	"time"
)

var (
	count      = 50
	size       = 1024
	sendBuffer = make([]byte, size*count)
	recvBuffer = make([]byte, size*count)
	done       = make(chan bool)
)

type testServer struct {
	t *testing.T
}

func (o *testServer) OmsgNewClient(conn net.Conn)        {}
func (o *testServer) OmsgClientClose(conn net.Conn)      {}
func (o *testServer) OmsgError(conn net.Conn, err error) { o.t.Fatal(err) }
func (o *testServer) OmsgData(conn net.Conn, cmd, ext uint16, data []byte) {
	// 收到客户端数据
	Send(conn, cmd, ext, data)
}

type testClient struct {
	t *testing.T
}

func (o *testClient) OmsgClose()          {}
func (o *testClient) OmsgError(err error) { o.t.Fatal(err) }
func (o *testClient) OmsgData(cmd, ext uint16, data []byte) {
	// 收到服务器数据
	if cmd != 1 {
		return
	}
	copy(recvBuffer[int(ext)*size:], data)
	if int(ext) == count-1 {
		done <- true
	}
}

func TestServerClient(t *testing.T) {
	runtime.GOMAXPROCS(runtime.NumCPU())
	log.SetFlags(log.Lshortfile)

	if _, err := crand.Read(sendBuffer); err != nil {
		t.Fatal(err)
	}
	// log.Println("\n" + hex.Dump(sendBuffer))

	// server
	go func() {
		s := NewServer(&testServer{t: t})
		log.Println(s.StartServer(":1234"))
	}()

	// client
	c := NewClient(&testClient{t: t})

	// connect
	for {
		if err := c.ConnectTimeout(":1234", time.Second*5); err == nil {
			break
		}
		time.Sleep(time.Second)
	}

	// send
	for i := 0; i < count; i++ {
		Send(c.Conn, 1, uint16(i), sendBuffer[i*size:(i+1)*size])
	}

	select {
	case <-time.After(time.Second * 5):
		t.Fatal("timeout")
	case <-done:
		if bytes.Compare(sendBuffer, recvBuffer) != 0 {
			t.Fail()
		}
	}
}

func TestByte(t *testing.T) {
	data := []byte("12345678123456781234")
	st := head{
		Sign: signWord,
		CRC:  crc(data),
		Cmd:  1,
		Ext:  2,
		Size: uint32(len(data)),
	}

	bs := bytes.NewBuffer(nil)
	binary.Write(bs, binary.LittleEndian, &st)
	bs.Write(data)
	// t.Log(hex.Dump(bs.Bytes()))

	bs2 := []byte{
		0x48, 0x4b, 0x81, 0x91, 0x01, 0x00, 0x02, 0x00, 0x14, 0x00, 0x00, 0x00, 0x31, 0x32, 0x33, 0x34,
		0x35, 0x36, 0x37, 0x38, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38, 0x31, 0x32, 0x33, 0x34,
	}

	if bytes.Compare(bs.Bytes(), bs2) != 0 {
		t.Fail()
	}
}

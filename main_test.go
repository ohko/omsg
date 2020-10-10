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
	s *Server
}

func (o *testServer) OnAccept(conn net.Conn) bool          { return true }
func (o *testServer) OnClientClose(conn net.Conn)          {}
func (o *testServer) OnRecvError(conn net.Conn, err error) { o.t.Fatal(err) }
func (o *testServer) OnData(conn net.Conn, cmd, ext uint16, data []byte) error {
	// 收到客户端数据
	return o.s.Send(conn, cmd, ext, data)
}

type testClient struct {
	t *testing.T
}

func (o *testClient) OnClose()              {}
func (o *testClient) OnRecvError(err error) { o.t.Fatal(err) }
func (o *testClient) OnData(cmd, ext uint16, data []byte) error {
	// 收到服务器数据
	if cmd != 1 {
		return nil
	}

	copy(recvBuffer[int(ext)*size:], data)
	if int(ext) == count-1 {
		done <- true
	}

	return nil
}

func TestServerClient(t *testing.T) {
	runtime.GOMAXPROCS(runtime.NumCPU())
	log.SetFlags(log.Lshortfile)

	crc := true

	if _, err := crand.Read(sendBuffer); err != nil {
		t.Fatal(err)
	}

	// server
	go func() {
		ts := &testServer{t: t}
		s, err := Listen("tcp", ":1234")
		if err != nil {
			log.Fatal(err)
		}
		ts.s = s
		log.Println(s.Run(ts, crc))
	}()

	// client
	tc := &testClient{t: t}

	// connect
	time.Sleep(time.Second)
	c, err := Dial("tcp", ":1234", tc, crc)
	if err != nil {
		log.Fatal(err)
	}

	// send
	for i := 0; i < count; i++ {
		c.Send(1, uint16(i), sendBuffer[i*size:(i+1)*size])
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

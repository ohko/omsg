package omsg

import (
	"bytes"
	crand "crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"log"
	"net"
	"runtime"
	"testing"
	"time"
)

func Test(t *testing.T) {
	runtime.GOMAXPROCS(runtime.NumCPU())
	log.SetFlags(log.Lshortfile)

	count := 50
	size := 1024
	sendBuffer := make([]byte, size*count)
	recvBuffer := make([]byte, size*count)
	if _, err := crand.Read(sendBuffer); err != nil {
		t.Fatal(err)
	}
	// log.Println("\n" + hex.Dump(sendBuffer))

	done := make(chan bool)

	// server
	go func() {
		s := NewServer()
		s.OnData = func(conn net.Conn, cmd, ext uint16, data []byte, err error) { // 收到客户端数据
			s.Send(conn, cmd, ext, data)
		}
		log.Println(s.StartServer(":1234"))
	}()

	// client
	c := NewClient()
	c.OnData = func(cmd, ext uint16, data []byte, err error) { // 收到服务器数据
		if cmd != 1 {
			return
		}
		copy(recvBuffer[int(ext)*size:], data)
		if int(ext) == count-1 {
			if bytes.Compare(sendBuffer, recvBuffer) == 0 {
				done <- true
			} else {
				t.Fail()
			}
		}
	}

	// connect
	for {
		if err := c.ConnectTimeout(":1234", time.Second*5); err == nil {
			break
		}
		time.Sleep(time.Second)
	}

	// send
	for i := 0; i < count; i++ {
		c.Send(1, uint16(i), sendBuffer[i*size:(i+1)*size])
	}

	select {
	case <-time.After(time.Second * 5):
		t.Fatal("timeout")
	case <-done:
	}
}

func Test1(t *testing.T) {

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
	t.Log(hex.Dump(bs.Bytes()))
}

package omsg

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"log"
	"net"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
)

var nClientSendSum int64 // 客户端发送数据和
var nClientRecvSum int64 // 客户端收到数据和

func Test(t *testing.T) {
	runtime.GOMAXPROCS(runtime.NumCPU())

	x := 50
	y := 500
	done := make(chan bool)

	// 创建服务器
	go func() {
		s := NewServer()
		s.OnData = func(c net.Conn, data []byte) { // 收到客户端数据
			s.Send(c, data)
		}
		log.Println(s.StartServer(":1234"))
	}()

	// 创建多个client
	for i := 0; i < x; i++ {

		// 异步
		go func(i int) {
			c := NewClient(false)
			c.OnData = func(data []byte) { // 收到服务器数据
				atomic.AddInt64(&nClientRecvSum, int64(len(data)))
			}
			for {
				if err := c.ConnectTimeout(":1234", time.Second*5); err == nil {
					break
				}
				time.Sleep(time.Second)
			}

			atomic.AddInt64(&nClientSendSum, int64(y))
			c.SendAsync(bytes.Repeat([]byte("."), y))
		}(i)

		// 同步
		go func(i int) {
			c := NewClient(true)
			for {
				if err := c.ConnectTimeout(":1234", time.Second*5); err == nil {
					break
				}
				time.Sleep(time.Second)
			}

			atomic.AddInt64(&nClientSendSum, int64(y))
			if data, err := c.SendSync(bytes.Repeat([]byte("."), y)); err == nil {
				atomic.AddInt64(&nClientRecvSum, int64(len(data)))
			} else {
				log.Println(err)
			}
		}(i)
	}

	go func() {
		for {
			time.Sleep(time.Microsecond)
			if nClientRecvSum == int64(x*2*y) {
				done <- true
			}
		}
	}()

	select {
	case <-time.After(time.Second * 5):
		log.Println("timeout")
	case <-done:
	}

	t.Log("客户端发送:", nClientSendSum)
	t.Log("客户端收到:", nClientRecvSum)
	if nClientSendSum != nClientRecvSum {
		t.Fail()
	}
}

func Test1(t *testing.T) {

	data := []byte("Hello world!")
	st := head{
		Sign: signWord,
		CRC:  crc(data),
		Size: uint32(len(data)),
	}

	bs := bytes.NewBuffer(nil)
	binary.Write(bs, binary.LittleEndian, &st)
	bs.Write(data)
	t.Log(hex.Dump(bs.Bytes()))
}

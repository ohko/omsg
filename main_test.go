package omsg

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
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
	log.SetFlags(log.Flags() | log.Lshortfile)
	runtime.GOMAXPROCS(runtime.NumCPU())

	x := 100
	done := make(chan bool)

	// 创建服务器
	go func() {
		s := NewServer()
		s.OnData = func(c net.Conn, counter uint32, data []byte, custom uint32) { // 收到客户端数据
			s.Send(c, counter, 0, data)
		}
		log.Println(s.StartServer(":1234"))
	}()

	// 创建多个client
	for i := 0; i < x; i++ {
		go func(i int) {
			c := NewClient()
			c.OnData = func(counter uint32, data []byte, custom uint32) { // 收到服务器数据
				atomic.AddInt64(&nClientRecvSum, int64(len(data)))
				if nClientRecvSum == int64(x*2*10) {
					done <- true
				}
			}
			for {
				if err := c.ConnectTimeout(":1234", time.Second*5); err == nil {
					break
				}
				time.Sleep(time.Second)
			}

			for j := 0; j < 10; j++ {
				atomic.AddInt64(&nClientSendSum, 1)
				c.SendAsync(0, []byte("."))
			}

			for j := 0; j < 10; j++ {
				atomic.AddInt64(&nClientSendSum, 1)
				if data, err := c.SendSync(123, []byte(".")); err == nil {
					atomic.AddInt64(&nClientRecvSum, int64(len(data)))
				} else {
					log.Println(err)
				}
			}

			if nClientRecvSum == int64(x*2*10) {
				done <- true
			}
		}(i)
	}

	select {
	case <-time.After(time.Second * 3):
		log.Println("timeout")
	case <-done:
	}

	fmt.Println("客户端发送:", nClientSendSum)
	fmt.Println("客户端收到:", nClientRecvSum)
}

func Test1(t *testing.T) {

	data := []byte("Hello world!")
	st := head{
		Sign:    0x4B48,
		CRC:     crc(data),
		Counter: 1,
		Size:    uint32(len(data)),
		Custom:  0x12345678,
	}

	bs := bytes.NewBuffer(nil)
	binary.Write(bs, binary.LittleEndian, &st)
	bs.Write(data)
	fmt.Println(hex.Dump(bs.Bytes()))
}

package omsg

import (
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

	x := 200
	key := []byte("1234567812345678")

	var cmdFromClient uint32 = 0x78563412
	var cmdFromServer uint32 = 0x21436587
	done := make(chan bool)

	// 创建服务器
	go func() {
		s := NewServer(key)
		s.OnData = func(conn net.Conn, cmd uint32, data []byte) { // 收到客户端数据
			s.Send(conn, cmdFromServer, data)
		}
		log.Println(s.StartServer(":1234"))
	}()

	// 创建多个client
	for i := 0; i < x; i++ {
		go func() {
			c := NewClient(key)
			c.OnConnect = func() { // 连接成功
				atomic.AddInt64(&nClientSendSum, 1)
				c.Send(cmdFromClient, []byte("."))
			}
			c.OnData = func(cmd uint32, data []byte) { // 收到服务器数据
				atomic.AddInt64(&nClientRecvSum, int64(len(data)))
				if nClientRecvSum == int64(x) {
					done <- true
				}
			}
			if err := c.ConnectTimeout(":1234", time.Second*5); err != nil {
				log.Println("[C] connect error:", err)
			}
		}()
	}

	<-done

	fmt.Println("客户端发送:", nClientSendSum)
	fmt.Println("客户端收到:", nClientRecvSum)
}

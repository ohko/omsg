package omsg

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
)

var s *Server
var nServerClient int64    // 成功连入的client数量
var nServerDataCount int64 // 服务器收到数据数量
var nServerClosed int64    // 成功断开的客户端数量
var nServerDataLen int64   // 服务器收到数据总大小
var nClientDataLen int64   // 客户端收到数据总大小

// 测试方案：并发发送0～100x1000，判断服务器收到的数据之和是否是预料值。
func Test(t *testing.T) {
	log.SetFlags(log.Flags() | log.Lshortfile)
	runtime.GOMAXPROCS(runtime.NumCPU())

	x := 100 // client数量
	y := 100 // 每个client发送多少次数据
	key := []byte("1234567812345678")
	var nNeed int64
	for i := 1; i <= x; i++ {
		for j := 1; j <= y; j++ {
			nNeed += int64(j)
		}
	}

	var wg sync.WaitGroup
	wg.Add(x)

	// 创建服务器
	s = NewServer(key,
		func(conn net.Conn, cmd uint32, data []byte) { // 收到客户端数据
			// fmt.Println(hex.Dump(data))
			atomic.AddInt64(&nServerDataCount, 1)
			atomic.AddInt64(&nServerDataLen, int64(len(data)))
			s.Send(conn, 0x21436587, data)
		},
		func(conn net.Conn) { // 新客户端通知
			atomic.AddInt64(&nServerClient, 1)
		},
		func(conn net.Conn) { // 客户端断开通知
			atomic.AddInt64(&nServerClosed, 1)
			wg.Done()
		},
	)
	go func() {
		s.StartServer("0.0.0.0:1234")
	}()

	// 创建多个client
	for i := 1; i <= x; i++ {
		go func(i int) {
			var c *Client
			c = NewClient(key,
				func() { // 连接成功
					// ...
				},
				func(cmd uint32, data []byte) { // 收到服务器数据
					atomic.AddInt64(&nClientDataLen, int64(len(data)))
					if len(data) == y { // 收到最后一条数据后断开
						c.Close()
					}
				},
				func() { // 服务器断开通知
					// ...
				},
				func() { // 重连通知
					// ...
				},
			)
			if err := c.Connect("0.0.0.0:1234"); err != nil {
				log.Fatalln(i, "[C] connect error:", err)
			}
			// 发数据
			for j := 1; j <= y; j++ {
				dd := bytes.Repeat([]byte("."), j)
				c.Send(0x78563412, dd)
			}
		}(i)
	}

	// 等待客户端断开
	wg.Wait()

	fmt.Println("预想数据:", nNeed)
	fmt.Println("客户端发送:", nClientDataLen)
	fmt.Println("服务器收到:", nServerDataLen)
	fmt.Println("成功断开:", nServerClosed)
}

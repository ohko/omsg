package omsg

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

var s *Server
var ch chan bool
var ai int64
var si int64
var ci int64
var dd string

const testData = "testData"

// Test ...
func Test(t *testing.T) {
	runtime.GOMAXPROCS(runtime.NumCPU())
	dd = strings.Join(make([]string, 1000*10000), ".")

	// go tool pprof xx cpu-profile.prof
	// f, err := os.Create("cpu-profile.prof")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// pprof.StartCPUProfile(f)
	main()
	// pprof.StopCPUProfile()
}

// 测试方案：并发发送0～100x1000，判断服务器收到的数据之和是否是预料值。
func main() {
	ch = make(chan bool)
	x := 1
	y := 100000
	for i := 1; i <= x; i++ {
		for j := 1; j <= y; j++ {
			ai += int64(i * j)
		}
	}

	// 创建一个服务器
	s := NewServer(onServerData, onNewClient, onServerClose)
	s.StartServer("0.0.0.0:1234")

	// 创建N个客户端
	for i := 1; i <= x; i++ {
		c := NewClient(onClientData, onClientClose)
		if err := c.Connect("0.0.0.0:1234", true, 5, 5); err != nil {
			log.Fatalln("[C] connect error:", err)
		}
		go func(i int) {
			for j := 1; j <= y; j++ {
				c.Send([]byte(fmt.Sprintf("%v,", i*j) + dd[:i*j]))
			}
		}(i)
	}

	<-ch
	fmt.Println("ai:", ai)
	fmt.Println("si:", si)
	fmt.Println("ci:", ci)
	if ai != si || ai != ci {
		log.Fatalln("测试失败")
	}
}

func onServerData(conn net.Conn, data []byte) {
	da := bytes.Split(data, []byte(","))
	n, _ := strconv.Atoi(string(da[0]))
	if n != len(da[1]) {
		log.Fatalln("[S] recv len err:", n, len(da[1]))
	}
	atomic.AddInt64(&si, int64(n))
	s.Send(conn, []byte(fmt.Sprintf("%v,", n)+dd[:n]))
}

func onNewClient(conn net.Conn) {
	// log.Println("[S] new client:", conn.RemoteAddr(), " new client.")
}

func onServerClose(conn net.Conn) {
	log.Println("[S]", conn.RemoteAddr(), " closed.")
}

func onClientData(data []byte) {
	da := bytes.Split(data, []byte(","))
	n, _ := strconv.Atoi(string(da[0]))
	if n != len(da[1]) {
		log.Fatalln("[C] recv len err:", n, len(da[1]))
	}
	atomic.AddInt64(&ci, int64(n))
	if ci == ai {
		ch <- true
	}
}

func onClientClose() {
	log.Println("[C] closed!")
}

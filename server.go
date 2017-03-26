package omsg

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"log"
	"net"
	"sync"
)

// Server ...
type Server struct {
	PrintDebugMsg bool
	server        net.Listener           // 用于服务器
	onData        func(net.Conn, []byte) // 数据回调
	onNewClient   func(net.Conn)         // 新客户端回调
	onClose       func(net.Conn)         // 客户端断开回调
	clientList    []net.Conn             // 客户端列表
	lock          sync.Mutex
}

// NewServer 创建
func NewServer(onData func(net.Conn, []byte), onNewClient func(net.Conn), onClose func(net.Conn)) *Server {
	return &Server{onData: onData, onNewClient: onNewClient, onClose: onClose}
}

// StartServer 启动服务
func (o *Server) StartServer(laddr string) error {
	defer func() { recover() }()
	var err error
	if o.server, err = net.Listen("tcp", laddr); err != nil {
		return err
	}
	go o.hListener(o.server)
	return nil
}

// 监听端口
func (o *Server) hListener(s net.Listener) {
	defer func() { recover() }()
	for {
		conn, err := s.Accept()
		if o.PrintDebugMsg {
			log.Printf("New client %v %v \n", conn.RemoteAddr(), err)
		}
		if err != nil {
			break
		}
		go o.hServer(conn)
	}
	if o.PrintDebugMsg {
		log.Printf("Server listener exit %v \n", s.Addr().String())
	}
}

// 接收数据
func (o *Server) hServer(conn net.Conn) {
	defer func() { recover() }()
	// 记录客户端
	o.lock.Lock()
	o.clientList = append(o.clientList, conn)
	o.lock.Unlock()

	// 新客户端回调
	if o.onNewClient != nil {
		o.onNewClient(conn)
	}
	defer conn.Close()

	// 接受数据缓存
	cache := new(bytes.Buffer)
	buf := make([]byte, 0x1024)
	var recvLen int
	var err error
	var needLen int

	// 接受数据
	for {
		if recvLen, err = conn.Read(buf); err != nil {
			break
		}

		// 写入缓存
		cache.Write(buf[:recvLen])

		if o.PrintDebugMsg {
			log.Printf("Server recv %v \n", hex.Dump(cache.Bytes()))
		}

		for {
			// 读取数据长度
			if needLen == 0 {
				// 头4字节是数据长度
				if cache.Len() < 4 {
					break
				}

				needLen = int(binary.LittleEndian.Uint32(cache.Next(4))) - 4
			}

			if o.PrintDebugMsg {
				log.Printf("Server left %v \n", hex.Dump(cache.Bytes()))
			}

			// 数据长度不够，继续读取
			if needLen > cache.Len() {
				if o.PrintDebugMsg {
					log.Printf("数据被拆包 %v / %v \n", cache.Len(), needLen)
				}
				break
			}

			// 数据回调
			if o.onData != nil {
				// o.onData(conn, cache.Next(needLen))
				_tmp := make([]byte, needLen)
				copy(_tmp, cache.Next(needLen))
				go o.onData(conn, _tmp)
				needLen = 0
			}
		}
	}

	// 断线
	if o.onClose != nil {
		o.onClose(conn)
	}

	// 从客户端列表移除
	o.lock.Lock()
	for k, v := range o.clientList {
		if v == conn {
			o.clientList = append(o.clientList[:k], o.clientList[k+1:]...)
			break
		}
	}
	o.lock.Unlock()
	if o.PrintDebugMsg {
		log.Printf("Client exit %v x %v \n", conn.LocalAddr(), conn.RemoteAddr())
	}
}

// SendToAll 向所有客户端发送数据
func (o *Server) SendToAll(x []byte) {
	defer func() { recover() }()
	o.lock.Lock()
	defer o.lock.Unlock()
	for _, v := range o.clientList {
		o.Send(v, x)
	}
}

// GetClients 获取所有客户端
func (o *Server) GetClients() []net.Conn {
	return o.clientList
}

// Send 向指定客户端发送数据
func (o *Server) Send(c net.Conn, x []byte) int {
	defer func() { recover() }()
	// 增加数据头，指定数据尺寸
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, uint32(len(x)+0x4))
	buf.Write(x)
	n, _ := c.Write(buf.Bytes())
	if o.PrintDebugMsg {
		log.Print("Send:", n, "\n", hex.Dump(buf.Bytes()))
	}
	if n >= 4 {
		return n - 4
	}
	return n
}

// GetClientTotal 获取客户端数量
func (o *Server) GetClientTotal() int {
	defer func() { recover() }()
	return len(o.clientList)
}

// Close 关闭服务器
func (o *Server) Close() {
	defer func() { recover() }()
	o.server.Close()
}

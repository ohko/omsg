package omsg

import (
	"bytes"
	"encoding/binary"
	"net"
	"sync"
	"time"
)

// Server 服务器
type Server struct {
	server      net.Listener           // 用于服务器
	onData      func(net.Conn, []byte) // 数据回调
	onNewClient func(net.Conn)         // 新客户端回调
	onClose     func(net.Conn)         // 客户端断开回调
	ClientList  map[net.Conn]*SClient  // 客户端列表
	lock        sync.Mutex
}

// SClient 服务器客户端对象
type SClient struct {
	Conn net.Conn  // 客户端连接
	Time time.Time // 连入时间
}

// NewServer 创建
func NewServer(onData func(net.Conn, []byte), onNewClient func(net.Conn), onClose func(net.Conn)) *Server {
	return &Server{
		onData: onData, onNewClient: onNewClient, onClose: onClose,
		ClientList: make(map[net.Conn]*SClient),
	}
}

// StartServer 启动服务
func (o *Server) StartServer(laddr string) error {
	var err error
	if o.server, err = net.Listen("tcp", laddr); err != nil {
		return err
	}
	go o.hListener(o.server)
	return nil
}

// 监听端口
func (o *Server) hListener(s net.Listener) {
	for {
		conn, err := s.Accept()
		if err != nil {
			break
		}
		go o.hServer(conn)
	}
}

// 接收数据
func (o *Server) hServer(conn net.Conn) {
	// 记录客户端
	o.lock.Lock()
	o.ClientList[conn] = &SClient{Conn: conn, Time: time.Now()}
	o.lock.Unlock()

	// 新客户端回调
	if o.onNewClient != nil {
		o.onNewClient(conn)
	}

	// 接受数据缓存
	cache := new(bytes.Buffer)
	buf := make([]byte, 0x100)
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

		for {
			// 读取数据长度
			if needLen == 0 {
				// 头4字节是数据长度
				if cache.Len() < 4 {
					break
				}

				needLen = int(binary.LittleEndian.Uint32(cache.Next(4))) - 4
			}

			// 数据长度不够，继续读取
			if needLen > cache.Len() {
				break
			}

			// 数据回调
			if o.onData != nil {
				o.onData(conn, cache.Next(needLen))
			} else {
				cache.Next(needLen)
			}
			needLen = 0
		}
	}

	// 断线
	if o.onClose != nil {
		o.onClose(conn)
	}

	// 从客户端列表移除
	o.lock.Lock()
	delete(o.ClientList, conn)
	o.lock.Unlock()
}

// SendToAll 向所有客户端发送数据
func (o *Server) SendToAll(x []byte) {
	o.lock.Lock()
	defer o.lock.Unlock()
	for _, v := range o.ClientList {
		o.Send(v.Conn, x)
	}
}

// Send 向指定客户端发送数据
func (o *Server) Send(c net.Conn, x []byte) (int, error) {
	// 增加数据头，指定数据尺寸
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(len(x)+0x4))

	if n, err := c.Write(buf[:]); err != nil {
		return n, err
	}

	return c.Write(x)
}

// Close 关闭服务器
func (o *Server) Close() {
	o.server.Close()
}

package omsg

import (
	"net"
	"sync"
	"time"
)

// Server 服务器
type Server struct {
	server      net.Listener                   // 用于服务器
	onData      func(net.Conn, uint32, []byte) // 数据回调
	onNewClient func(net.Conn)                 // 新客户端回调
	onClose     func(net.Conn)                 // 客户端断开回调
	ClientList  map[net.Conn]*SClient          // 客户端列表
	lock        sync.Mutex
	crypt       *crypt
}

// SClient 服务器客户端对象
type SClient struct {
	Conn net.Conn  // 客户端连接
	Time time.Time // 连入时间
}

// NewServer 创建
func NewServer(key []byte, onData func(net.Conn, uint32, []byte), onNewClient func(net.Conn), onClose func(net.Conn)) *Server {
	o := &Server{
		onData: onData, onNewClient: onNewClient, onClose: onClose,
		ClientList: make(map[net.Conn]*SClient),
	}
	if key != nil {
		o.crypt = newCrypt(key)
	}
	return o
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

	recv(o.crypt, conn, o.onData, nil)

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
func (o *Server) SendToAll(cmd uint32, x []byte) {
	o.lock.Lock()
	defer o.lock.Unlock()
	for _, v := range o.ClientList {
		o.Send(v.Conn, cmd, x)
	}
}

// Send 向指定客户端发送数据
func (o *Server) Send(c net.Conn, cmd uint32, data []byte) (int, error) {
	return send(o.crypt, c, cmd, data)
}

// Close 关闭服务器
func (o *Server) Close() {
	o.server.Close()
}

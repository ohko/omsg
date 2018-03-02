package omsg

import (
	"net"
	"time"
)

// Client ...
type Client struct {
	client         net.Conn                      // 用户客户端
	onConnect      func()                        // 成功连接回调
	onData         func(cmd uint32, data []byte) // 收到命令行回调
	onClose        func()                        // 连接断开回调
	onReconnect    func()                        // 重连回调
	serverAddr     string                        // 服务器地址
	ReConnect      bool                          // 断线重连
	ConnectTimeout time.Duration                 // 连线超时时间（秒）
	ReConnectWait  time.Duration                 // 重连时间间隔（秒）
	crypt          *crypt
}

// NewClient 创建客户端
func NewClient(key []byte, onConnect func(), onData func(cmd uint32, data []byte), onClose func(), onReconnect func()) *Client {
	o := &Client{onConnect: onConnect, onData: onData, onClose: onClose}
	o.ReConnect = false
	o.ConnectTimeout = time.Second
	o.ReConnectWait = time.Second
	if key != nil {
		o.crypt = newCrypt(key)
	}
	return o
}

// Connect 连接到服务器
func (o *Client) Connect(address string) error {
	o.serverAddr = address
	return o.connect(false)
}

func (o *Client) connect(re bool) error {
	for {
		var err error
		if o.client, err = net.DialTimeout("tcp", o.serverAddr, o.ConnectTimeout); err == nil {
			if o.onConnect != nil {
				o.onConnect()
			}
			break
		}
		if !o.ReConnect {
			return err
		}
		if re && o.onReconnect != nil {
			o.onReconnect()
		}
		time.Sleep(o.ReConnectWait)
	}
	go o.hClient()
	return nil
}

// 监听数据
func (o *Client) hClient() {
	recv(o.crypt, o.client, nil, o.onData)

	o.client.Close()
	if o.ReConnect {
		o.connect(true)
	}
	if o.onClose != nil {
		o.onClose()
	}
}

// Send 向服务器发送数据
func (o *Client) Send(cmd uint32, data []byte) (int, error) {
	tmp := make([]byte, len(data))
	copy(tmp, data)
	return send(o.crypt, o.client, cmd, tmp)
}

// Close 关闭链接
func (o *Client) Close() {
	o.client.Close()
}

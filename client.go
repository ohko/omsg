package omsg

import (
	"net"
	"time"
)

// Client ...
type Client struct {
	client    net.Conn                      // 用户客户端
	OnConnect func()                        // 成功连接回调
	OnData    func(cmd uint32, data []byte) // 收到命令行回调
	OnClose   func()                        // 连接断开回调
	crypt     *crypt
}

// NewClient 创建客户端
func NewClient(key []byte) *Client {
	o := &Client{}
	if key != nil {
		o.crypt = newCrypt(key)
	}
	return o
}

// Connect 连接到服务器
func (o *Client) Connect(address string) error {
	var err error
	if o.client, err = net.Dial("tcp", address); err != nil {
		return err
	}
	if o.OnConnect != nil {
		o.OnConnect()
	}
	go o.hClient()
	return nil
}

// ConnectTimeout 连接到服务器
func (o *Client) ConnectTimeout(address string, timeout time.Duration) error {
	var err error
	if o.client, err = net.DialTimeout("tcp", address, timeout); err != nil {
		return err
	}
	if o.OnConnect != nil {
		o.OnConnect()
	}
	go o.hClient()
	return nil
}

// 监听数据
func (o *Client) hClient() {
	recv(o.crypt, o.client, nil, o.OnData)

	o.client.Close()
	if o.OnClose != nil {
		o.OnClose()
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

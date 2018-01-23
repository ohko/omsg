package omsg

import (
	"net"
	"time"
)

// Client ...
type Client struct {
	client         net.Conn      // 用户客户端
	onData         func([]byte)  // 收到命令行回调
	onClose        func()        // 链接断开回调
	onReconnect    func()        // 重连回调
	serverAddr     string        // 服务器地址
	reconnect      bool          // 断线重连
	connectTimeout time.Duration // 连线超时时间（秒）
	reconnectWait  time.Duration // 重连时间间隔（秒）
	crypt          *crypt
}

// NewClient 创建客户端
func NewClient(key []byte, onData func([]byte), onClose func(), onReconnect func()) *Client {
	return &Client{onData: onData, onClose: onClose, crypt: newCrypt(key)}
}

// Connect 连接到服务器
func (o *Client) Connect(address string, reconnect bool, connectTimeout time.Duration, reconnectWait time.Duration) error {
	o.serverAddr = address
	o.reconnect = reconnect
	o.connectTimeout = connectTimeout
	o.reconnectWait = reconnectWait

	return o.connect(false)
}

func (o *Client) connect(re bool) error {
	for {
		var err error
		if o.client, err = net.DialTimeout("tcp", o.serverAddr, o.connectTimeout); err == nil {
			break
		}
		if !o.reconnect {
			return err
		}
		if re && o.onReconnect != nil {
			o.onReconnect()
		}
		time.Sleep(o.reconnectWait)
	}
	go o.hClient()
	return nil
}

// 监听数据
func (o *Client) hClient() {
	recv(o.crypt, o.client, nil, o.onData)

	o.client.Close()
	if o.reconnect {
		o.connect(true)
	}
	if o.onClose != nil {
		o.onClose()
	}
}

// Send 向服务器发送数据
func (o *Client) Send(data []byte) (int, error) {
	return send(o.crypt, o.client, data)
}

// Close 关闭链接
func (o *Client) Close() {
	o.client.Close()
}

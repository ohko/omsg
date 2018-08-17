package omsg

import (
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// Client ...
type Client struct {
	client   net.Conn               // 用户客户端
	counter  uint32                 // 计数器
	OnData   ClientCallback         // 收到命令行回调
	OnClose  func()                 // 连接断开回调
	sync     map[uint32]chan []byte // 同步请求
	syncLock sync.RWMutex
}

// NewClient 创建客户端
func NewClient() *Client {
	o := &Client{sync: make(map[uint32]chan []byte)}
	return o
}

// Connect 连接到服务器
func (o *Client) Connect(address string) error {
	return o.ConnectTimeout(address, 0)
}

// ConnectTimeout 连接到服务器
func (o *Client) ConnectTimeout(address string, timeout time.Duration) error {
	var err error
	if o.client, err = net.DialTimeout("tcp", address, timeout); err != nil {
		return err
	}
	go o.hClient()
	return nil
}

// 监听数据
func (o *Client) hClient() {
	recv(o.client, nil, o.SelfOnData)

	o.client.Close()
	if o.OnClose != nil {
		o.OnClose()
	}
}

// SendAsync 异步向服务器发送数据
func (o *Client) SendAsync(custom uint32, data []byte) (int, error) {
	return send(o.client, 0, custom, data)
}

// SendSync 同步向服务器发送数据
func (o *Client) SendSync(custom uint32, data []byte) ([]byte, error) {

	counter := atomic.AddUint32(&o.counter, 1)

	if _, err := send(o.client, counter, custom, data); err != nil {
		return nil, err
	}

	o.syncLock.Lock()
	ch := make(chan []byte)
	o.sync[counter] = ch
	o.syncLock.Unlock()

	bs := <-ch

	o.syncLock.Lock()
	delete(o.sync, counter)
	o.syncLock.Unlock()
	return bs, nil
}

// SelfOnData 收到服务器数据
func (o *Client) SelfOnData(counter uint32, data []byte, custom uint32) {
	if counter == 0 {
		if o.OnData != nil {
			o.OnData(counter, data, custom)
		}
		return
	}

	o.syncLock.RLock()
	defer o.syncLock.RUnlock()
	if v, ok := o.sync[counter]; ok {
		v <- data
	}
}

// Close 关闭链接
func (o *Client) Close() {
	o.client.Close()
}

package omsg

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"log"
	"net"
	"time"
)

// Client ...
type Client struct {
	PrintDebugMsg  bool
	client         net.Conn     // 用户客户端
	onData         func([]byte) // 收到命令行回调
	onClose        func()       // 链接断开回调
	status         int          // 状态：0=未连接／1=已连接
	serverAddr     string       // 服务器地址
	reconnect      bool         // 断线重连
	connectTimeout int          // 连线超时时间（秒）
	reconnectWait  int          // 重连时间间隔（秒）
}

// NewClient 创建客户端
func NewClient(onData func([]byte), onClose func()) *Client {
	return &Client{onData: onData, onClose: onClose}
}

// Connect 连接到服务器
func (o *Client) Connect(address string, reconnect bool, connectTimeout int, reconnectWait int) error {
	defer func() { recover() }()
	o.serverAddr = address
	o.reconnect = reconnect
	o.connectTimeout = connectTimeout
	o.reconnectWait = reconnectWait

	return o.reConnect()
}

func (o *Client) reConnect() error {
	defer func() { recover() }()

	for {
		var err error
		o.client, err = net.DialTimeout("tcp", o.serverAddr, time.Second*time.Duration(o.connectTimeout))
		if err == nil {
			break
		}
		if !o.reconnect {
			return err
		}
		log.Println("reconnect...", o.serverAddr, err)
		time.Sleep(time.Second * time.Duration(o.reconnectWait))
	}
	o.status = 1
	go o.hClient()
	return nil
}

// 监听数据
func (o *Client) hClient() {
	defer func() { recover() }()
	if o.PrintDebugMsg {
		log.Printf("Connect server %v -> %v \n", o.client.LocalAddr(), o.client.RemoteAddr())
	}

	// 数据缓存
	cache := new(bytes.Buffer)
	buf := make([]byte, 0x1024)
	var recvLen int
	var err error
	var needLen int

	for {
		if recvLen, err = o.client.Read(buf); err != nil {
			break
		}

		// 写入缓存
		cache.Write(buf[:recvLen])

		if o.PrintDebugMsg {
			log.Printf("Client recv %v \n", hex.Dump(cache.Bytes()))
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

			// 数据长度不够，继续读取
			if needLen > cache.Len() {
				break
			}

			// 数据回调
			if o.onData != nil {
				// o.onData(cache.Next(needLen))
				_tmp := make([]byte, needLen)
				copy(_tmp, cache.Next(needLen))
				go o.onData(_tmp)
				needLen = 0
			}
		}
	}
	o.status = 0
	if o.reconnect {
		o.reConnect()
	}
	if o.onClose != nil {
		o.onClose()
	}
	if o.PrintDebugMsg {
		log.Printf("Connect exit %v x %v \n", o.client.LocalAddr(), o.client.RemoteAddr())
	}
}

// Send 向服务器发送数据
func (o *Client) Send(x []byte) int {
	if o.status == 0 {
		return 0
	}
	defer func() { recover() }()
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, uint32(len(x)+0x4))
	buf.Write(x)
	n, _ := o.client.Write(buf.Bytes())
	if o.PrintDebugMsg {
		log.Print("Send:", n, "\n", hex.Dump(buf.Bytes()))
	}
	if n >= 4 {
		return n - 4
	}
	return n
}

// Close 关闭链接
func (o *Client) Close() {
	defer func() { recover() }()
	o.client.Close()
}

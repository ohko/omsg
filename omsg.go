package omsg

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
)

type head struct {
	Sign    uint16 // 2数据标志 HK
	CRC     uint16 // 2简单crc校验值
	Counter uint32 // 4计数器
	Size    uint32 // 4数据大小
	Custom  uint32 // 4自定义
}

// ServerCallback 服务器端数据回调函数
type ServerCallback func(c net.Conn, counter uint32, data []byte, custom uint32)

// ClientCallback 客户端数据回调函数
type ClientCallback func(counter uint32, data []byte, custom uint32)

var headSize = binary.Size(head{})
var sendLock sync.Mutex

const (
	sign = 0x4B48 // HK
)

func send(conn net.Conn, counter uint32, custom uint32, data []byte) (int, error) {
	sendLock.Lock()
	defer sendLock.Unlock()

	// head
	h := head{Sign: sign, CRC: crc(data), Counter: counter, Size: uint32(len(data)), Custom: custom}

	hbs := bytes.NewBuffer(nil)
	binary.Write(hbs, binary.LittleEndian, h)
	if _, err := conn.Write(hbs.Bytes()); err != nil {
		return 0, err
	}

	return conn.Write(data)
}

func recv(conn net.Conn, sCallback ServerCallback, cCallback ClientCallback) {
	cache := bytes.NewBuffer(nil)
	buf := make([]byte, 0x100)
	var recvLen int
	var err error
	needHead := new(head)

	for {
		if recvLen, err = conn.Read(buf); err != nil {
			break
		}

		// 写入缓存
		cache.Write(buf[:recvLen])

		for {
			// 数据不足头数据长度
			if cache.Len() <= headSize {
				break
			}

			// 读取数据头
			binary.Read(bytes.NewBuffer(cache.Bytes()), binary.LittleEndian, needHead)
			if needHead.Sign != sign {
				fmt.Println("Header err")
				return
			}

			// 数据长度不够，继续读取
			if cache.Len() < int(needHead.Size) {
				break
			}

			// 数据回调
			binary.Read(cache, binary.LittleEndian, needHead)
			tmp := make([]byte, needHead.Size)
			cache.Read(tmp)
			if needHead.CRC == crc(tmp) {
				if sCallback != nil {
					go sCallback(conn, needHead.Counter, tmp, needHead.Custom)
				} else if cCallback != nil {
					go cCallback(needHead.Counter, tmp, needHead.Custom)
				}
			} else {
				fmt.Println("crc error")
			}
		}
	}
}

func crc(data []byte) uint16 {
	size := len(data)
	crc := 0xFFFF
	for i := 0; i < size; i++ {
		crc = (crc >> 8) ^ int(data[i])
		for j := 0; j < 8; j++ {
			flag := crc & 0x0001
			crc >>= 1
			if flag == 1 {
				crc ^= 0xA001
			}
		}
	}
	return uint16(crc)
}

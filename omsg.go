package omsg

import (
	"bytes"
	"encoding/binary"
	"net"
)

type head struct {
	Sign        uint16 // 2数据标志
	Cmd         uint8  // 1指令
	CRC         uint16 // 2简单crc校验值
	SizeOrigin  uint32 // 4原始数据大小
	SizeEncrypt uint32 // 4加密后数据大小
}

var headSize = binary.Size(head{})

const (
	sign     = 0x4B48 // HK
	cmdLogin = iota   // 登陆
	cmdData           // 数据
)

func send(c *crypt, conn net.Conn, data []byte) (int, error) {
	// head
	h := head{Sign: sign, Cmd: cmdData, CRC: 0, SizeOrigin: uint32(len(data))}
	if h.SizeOrigin%16 != 0 {
		h.SizeEncrypt = h.SizeOrigin + 16 - (h.SizeOrigin % 16)
	}

	// fix data length
	if h.SizeOrigin != h.SizeEncrypt {
		data = append(data, bytes.Repeat([]byte{0}, int(h.SizeEncrypt-h.SizeOrigin))...)
	}

	// encrypt
	c.Encrypt(data)

	// crc
	h.CRC = crc(data)

	hbs := bytes.NewBuffer(nil)
	binary.Write(hbs, binary.LittleEndian, h)
	if n, err := conn.Write(hbs.Bytes()); err != nil {
		return n, err
	}
	return conn.Write(data)
}

func recv(c *crypt, conn net.Conn, sCallback func(net.Conn, []byte), cCallback func([]byte)) {
	cache := new(bytes.Buffer)
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
			// 读取数据长度
			if needHead.SizeEncrypt == 0 {
				// 头数据长度
				if cache.Len() < headSize {
					break
				}

				bs := bytes.NewBuffer(cache.Next(headSize))
				binary.Read(bs, binary.LittleEndian, needHead)
			}

			if needHead.Sign != sign {
				return
			}

			// 数据长度不够，继续读取
			if int(needHead.SizeEncrypt) > cache.Len() {
				break
			}

			// 数据回调
			tmp := cache.Next(int(needHead.SizeEncrypt))
			if needHead.CRC == crc(tmp) {
				c.Decrypt(tmp)
				if sCallback != nil {
					sCallback(conn, tmp[:int(needHead.SizeOrigin)])
				} else if cCallback != nil {
					cCallback(tmp[:int(needHead.SizeOrigin)])
				} else {
					cache.Next(int(needHead.SizeEncrypt))
				}
			}
			needHead.SizeEncrypt = 0
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

package omsg

import (
	"bytes"
	"encoding/binary"
	"log"
	"net"
)

type head struct {
	Sign        uint16 // 2数据标志
	CRC         uint16 // 2简单crc校验值
	SizeOrigin  uint32 // 4原始数据大小
	SizeEncrypt uint32 // 4加密后数据大小
	Cmd         uint32 // 4指令
}

var headSize = binary.Size(head{})

const (
	sign = 0x4B48 // HK
)

func send(c *crypt, conn net.Conn, cmd uint32, data []byte) (int, error) {
	// head
	h := head{Sign: sign, CRC: 0, SizeOrigin: uint32(len(data)), Cmd: cmd}
	if c != nil { // encrypt
		data = c.encrypt(data)
		h.SizeEncrypt = uint32(len(data))
	} else {
		h.SizeEncrypt = h.SizeOrigin
	}

	// crc
	h.CRC = crc(data)

	hbs := bytes.NewBuffer(nil)
	binary.Write(hbs, binary.LittleEndian, h)
	if n, err := conn.Write(hbs.Bytes()); err != nil {
		return n, err
	}
	if n, err := conn.Write(data); err != nil {
		return n, err
	}
	return int(h.SizeOrigin), nil
}

func recv(c *crypt, conn net.Conn, sCallback func(net.Conn, uint32, []byte), cCallback func(uint32, []byte)) {
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
				if c != nil {
					tmp = c.decrypt(tmp)
				}
				if sCallback != nil {
					sCallback(conn, needHead.Cmd, tmp[:int(needHead.SizeOrigin)])
				} else if cCallback != nil {
					cCallback(needHead.Cmd, tmp[:int(needHead.SizeOrigin)])
				} else {
					cache.Next(int(needHead.SizeEncrypt))
				}
			} else {
				log.Println("crc error")
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

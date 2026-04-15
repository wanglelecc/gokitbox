package uHash

import (
	"encoding/hex"
	"fmt"
	"strings"
)

// CRC16Modbus 计算 Modbus 协议的 CRC16 校验值
//
// 返回高 8 位在前的 uint16 校验值
//
// 使用示例：
//
//	crc := uHash.CRC16Modbus([]byte{0x01, 0x03, 0x00, 0x00, 0x00, 0x0A})
//	// crc 为对应的校验值（示例值取决于输入数据）
func CRC16Modbus(data []byte) uint16 {
	crc := uint16(0xFFFF)
	for _, b := range data {
		crc ^= uint16(b)
		for i := 0; i < 8; i++ {
			if crc&0x0001 > 0 {
				crc = (crc >> 1) ^ 0xA001
			} else {
				crc >>= 1
			}
		}
	}
	return (crc << 8) | (crc >> 8)
}

// CRC16ModbusFromHex 从 16 进制字符串解析数据并计算 Modbus CRC16 校验值
//
// 使用示例：
//
//	crc, err := uHash.CRC16ModbusFromHex("01030000000A")
//	// crc 为对应的校验值（示例值取决于输入数据）
func CRC16ModbusFromHex(hexStr string) (uint16, error) {
	data, err := hex.DecodeString(hexStr)
	if err != nil {
		return 0, err
	}
	return CRC16Modbus(data), nil
}

// Crc16Encode 对 16 进制字符串进行 CRC16-IBM 编码，返回 4 位大写 16 进制 CRC 值
//
// 如果 hexStr 长度为奇数，会自动在前面补 0
//
// 使用示例：
//
//	crc, err := uHash.Crc16Encode("0103")
//	// crc = "0C86"（示例值，取决于输入数据）
func Crc16Encode(hexStr string) (string, error) {
	s := strings.TrimSpace(hexStr)
	if len(s)%2 != 0 {
		s = "0" + s
	}
	data, err := hex.DecodeString(s)
	if err != nil {
		return "", err
	}
	crc := crc16ibm(data)
	return fmt.Sprintf("%04X", crc), nil
}

// reverseByte 反转字节中的位顺序（bit0 与 bit7 互换，bit1 与 bit6 互换，以此类推）
func reverseByte(b byte) byte {
	var res byte = 0
	for i := 0; i < 8; i++ {
		if (b & (1 << i)) != 0 {
			res |= (1 << (7 - i))
		}
	}
	return res
}

// reverseBytesWithSwap 反转字节切片中每个字节的位顺序，并对切片进行首尾对调
func reverseBytesWithSwap(s []byte) []byte {
	m, n := 0, len(s)-1
	for m <= n {
		if m == n {
			s[m] = reverseByte(s[m])
			break
		}
		s[m], s[n] = reverseByte(s[n]), reverseByte(s[m])
		m++
		n--
	}
	return s
}

// crc16ibm 计算 IBM 标准的 CRC16 校验值
func crc16ibm(data []byte) uint16 {
	crc := uint16(0xffff)
	for _, b := range data {
		b = reverseByte(b)
		crc ^= uint16(b) << 8
		for i := 0; i < 8; i++ {
			if crc&0x8000 != 0 {
				crc = (crc<<1)&0xffff ^ 0x8005
			} else {
				crc = (crc << 1) & 0xffff
			}
		}
	}
	ret := []byte{byte(crc & 0xff), byte((crc >> 8) & 0xff)}
	ret = reverseBytesWithSwap(ret)
	return uint16(ret[0]) | uint16(ret[1])<<8
}

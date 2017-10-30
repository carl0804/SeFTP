package Controller

import (
	"net"
	"log"
	"fmt"
	"io"
	"encoding/binary"
	"crypto/rand"
)

type TCPController struct {
	ServerAddr string
	Conn       net.Conn
	Passwd     [32]byte
}

func (tcpCon *TCPController) EstabConn() {
	conn, err := net.Dial("tcp", tcpCon.ServerAddr)
	if err != nil {
		log.Println(err)
	}
	tcpCon.Conn = conn
	log.Println("Conn Established.")
}

func (tcpCon *TCPController) CloseConn() {
	tcpCon.Conn.Close()
	fmt.Println("Dial closed.")
}

func (tcpCon *TCPController) SendByte(data []byte) {
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		log.Println(err.Error())
	}

	encByte := GCMEncrypter(data, tcpCon.Passwd, nonce)
	bs := make([]byte, 2)
	binary.LittleEndian.PutUint16(bs, uint16(len(encByte)))
	finalPac := append(append(nonce, bs...), encByte...)
	tcpCon.Conn.Write(finalPac)
}

func (tcpCon *TCPController) GetByte() ([]byte, error) {
	buf := make([]byte, 65550)
	rLen := 0
	n, rErr := tcpCon.Conn.Read(buf)
	buf = buf[:n]
	log.Println("Package Length Received: ", n)
	rLen += n

	if rErr == nil {
		lth := buf[12:14]
		//log.Println(lth)
		length := binary.LittleEndian.Uint16(lth)
		log.Println("Package Length Defined: ", length)
		for {
			if rLen < int(length)+14 {
				subbuf := make([]byte, 65550)
				n, rErr := tcpCon.Conn.Read(subbuf)
				subbuf = subbuf[:n]
				if rErr == nil {
					log.Println("Package Length Received: ", n)
					buf = append(buf, subbuf...)
					rLen += n
				}
				continue
			} else {
				log.Println("RECV PACKAGE COMPLETE")
				break
			}
		}
		//log.Println("BUF: ", buf)
		nonce, buf := buf[:12], buf[14:]
		data, buf := buf[:length], buf[length:]
		decData, err := GCMDecrypter(data, tcpCon.Passwd, nonce)
		return decData, err
	}
	return nil, rErr
}

func (tcpCon *TCPController) SendText(text string) {
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		log.Println(err.Error())
	}

	encByte := GCMEncrypter([]byte(text), tcpCon.Passwd, nonce)
	bs := make([]byte, 2)
	binary.LittleEndian.PutUint16(bs, uint16(len(encByte)))
	finalPac := append(append(nonce, bs...), encByte...)
	tcpCon.Conn.Write(finalPac)
}

func (tcpCon *TCPController) GetText() (string, error) {
	buf := make([]byte, 4096)
	_, rErr := tcpCon.Conn.Read(buf)

	if rErr == nil {
		nonce, buf := buf[:12], buf[12:]
		lth, buf := buf[:2], buf[2:]
		length := binary.LittleEndian.Uint16(lth)
		data, buf := buf[:length], buf[length:]
		decData, err := GCMDecrypter(data, tcpCon.Passwd, nonce)
		return string(decData), err
	}
	return "", rErr
}

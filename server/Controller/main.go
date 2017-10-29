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
	Listener       net.Listener
	Passwd     [32]byte
}

func (tcpCon *TCPController) EstabListener() {
	ln, err := net.Listen("tcp", tcpCon.ServerAddr)
	if err != nil {
		panic(err)
	}
	tcpCon.Listener = ln
	log.Println("Listener Established.")
}

func (tcpCon *TCPController) CloseListener() {
	tcpCon.Listener.Close()
	fmt.Println("Listener closed.")
}

func (tcpCon *TCPController) SendByte(conn net.Conn, data []byte) {
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		panic(err.Error())
	}

	encByte := GCMEncrypter(data, tcpCon.Passwd, nonce)
	bs := make([]byte, 2)
	binary.LittleEndian.PutUint16(bs, uint16(len(encByte)))
	finalPac := append(append(nonce, bs...), encByte...)
	conn.Write(finalPac)
}

func (tcpCon *TCPController) GetByte(conn net.Conn) ([]byte, error) {
	buf := make([]byte, 4096)
	_, rErr := conn.Read(buf)

	if rErr == nil {
		nonce, buf := buf[:12], buf[12:]
		lth, buf := buf[:2], buf[2:]
		length := binary.LittleEndian.Uint16(lth)
		data, buf := buf[:length], buf[length:]
		decData := GCMDecrypter(data, tcpCon.Passwd, nonce)
		return decData, nil
	}
	return nil, rErr
}

func (tcpCon *TCPController) SendText(conn net.Conn, text string) {
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		panic(err.Error())
	}

	encByte := GCMEncrypter([]byte(text), tcpCon.Passwd, nonce)
	bs := make([]byte, 2)
	binary.LittleEndian.PutUint16(bs, uint16(len(encByte)))
	finalPac := append(append(nonce, bs...), encByte...)
	conn.Write(finalPac)
}

func (tcpCon *TCPController) GetText(conn net.Conn) (string, error) {
	buf := make([]byte, 4096)
	_, rErr := conn.Read(buf)

	if rErr == nil {
		nonce, buf := buf[:12], buf[12:]
		lth, buf := buf[:2], buf[2:]
		length := binary.LittleEndian.Uint16(lth)
		data, buf := buf[:length], buf[length:]
		decData := GCMDecrypter(data, tcpCon.Passwd, nonce)
		return string(decData), nil
	}
	return "", rErr
}
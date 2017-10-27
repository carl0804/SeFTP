package Controller

import (
	"net"
	"log"
	"fmt"
	"io"
	"encoding/binary"
	"crypto/rand"
)

type SeFTPController struct {
	ServerAddr string
	Listener       net.Listener
	Passwd     [32]byte
}

func (seftpCon *SeFTPController) EstabListener() {
	ln, err := net.Listen("tcp", seftpCon.ServerAddr)
	if err != nil {
		panic(err)
	}
	seftpCon.Listener = ln
	log.Println("Listener Established.")
}

func (seftpCon *SeFTPController) CloseListener() {
	seftpCon.Listener.Close()
	fmt.Println("Listener closed.")
}

func (seftpCon *SeFTPController) SendText(conn net.Conn, text string) {
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		panic(err.Error())
	}

	encByte := GCMEncrypter([]byte(text), seftpCon.Passwd, nonce)
	//encLength := GCMEncrypter(strconv.Itoa(len(encByte)), "AES256Key-32Characters1234567890", nonce)
	//fmt.Println("Nonce:", nonce)
	bs := make([]byte, 2)
	binary.LittleEndian.PutUint16(bs, uint16(len(encByte)))
	//fmt.Println("Length:", bs)
	//fmt.Println("Data:", encByte)
	finalPac := append(append(nonce, bs...), encByte...)
	conn.Write(finalPac)
}

func (seftpCon *SeFTPController) GetText(conn net.Conn) (string, error) {
	buf := make([]byte, 4096)
	_, rErr := conn.Read(buf)

	if rErr == nil {
		//fmt.Println(buf[:])
		nonce, buf := buf[:12], buf[12:]
		//fmt.Println("Nonce:", nonce)
		lth, buf := buf[:2], buf[2:]
		length := binary.LittleEndian.Uint16(lth)
		//fmt.Println("Length:", length)
		data, buf := buf[:length], buf[length:]
		//fmt.Println("Data:", data)
		//fmt.Println("Remain:", buf)
		decData := GCMDecrypter(data, seftpCon.Passwd, nonce)
		//fmt.Println("Package Length:", rLen)
		//fmt.Println("decData:", string(decData))
		return string(decData), nil
	}
	return "", rErr
}

type SubFTPController struct {
	ServerAddr string
	Listener       net.Listener
	Passwd     [32]byte
}

func (subftpCon *SubFTPController) EstabListener() {
	ln, err := net.Listen("tcp", subftpCon.ServerAddr)
	if err != nil {
		panic(err)
	}
	subftpCon.Listener = ln
	log.Println("Listener Established.")
}

func (subftpCon *SubFTPController) CloseListener() {
	subftpCon.Listener.Close()
	fmt.Println("Listener closed.")
}

func (subftpCon *SubFTPController) SendByte(conn net.Conn, data []byte) {
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		panic(err.Error())
	}

	encByte := GCMEncrypter(data, subftpCon.Passwd, nonce)
	//encLength := GCMEncrypter(strconv.Itoa(len(encByte)), "AES256Key-32Characters1234567890", nonce)
	//fmt.Println("Nonce:", nonce)
	bs := make([]byte, 2)
	binary.LittleEndian.PutUint16(bs, uint16(len(encByte)))
	//fmt.Println("Length:", bs)
	//fmt.Println("Data:", encByte)
	finalPac := append(append(nonce, bs...), encByte...)
	conn.Write(finalPac)
}
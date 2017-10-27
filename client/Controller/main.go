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
	Conn       net.Conn
	Passwd     [32]byte
}

func (seftpCon *SeFTPController) EstabConn() {
	conn, err := net.Dial("tcp", seftpCon.ServerAddr)
	if err != nil {
		panic(err)
	}
	seftpCon.Conn = conn
	log.Println("Conn Established.")
}

func (seftpCon *SeFTPController) CloseConn() {
	seftpCon.Conn.Close()
	fmt.Println("Dial closed.")
}

func (seftpCon *SeFTPController) SendText(text string) {
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
	seftpCon.Conn.Write(finalPac)
}

func (seftpCon *SeFTPController) GetText() (string, error) {
	buf := make([]byte, 4096)
	_, rErr := seftpCon.Conn.Read(buf)

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

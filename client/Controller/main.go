package Controller

import (
	"crypto/rand"
	"encoding/binary"
	"io"
	"log"
	"net"
	"github.com/xtaci/smux"
)

type TCPController struct {
	ServerAddr string
	Conn       net.Conn
	Smux       *smux.Session
	Stream     *smux.Stream
	Passwd     [32]byte
}

func (tcpCon *TCPController) EstabConn() {
	conn, err := net.Dial("tcp", tcpCon.ServerAddr)
	if err != nil {
		log.Println(err)
	}
	tcpCon.Conn = conn
	log.Println("Conn Established.")

	session, err := smux.Client(conn, nil)
	if err != nil {
		log.Println(err)
	}
	tcpCon.Smux = session
	log.Println("Smux Established.")

	// Open a new stream
	stream, err := session.OpenStream()
	if err != nil {
		panic(err)
	}
	tcpCon.Stream = stream
	log.Println("Stream Established.")
}

func (tcpCon *TCPController) CloseConn() {
	tcpCon.Stream.Close()
	tcpCon.Smux.Close()
	tcpCon.Conn.Close()
	log.Println("Dial closed.")
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
	tcpCon.Stream.Write(finalPac)
}

func (tcpCon *TCPController) GetByte(exbuf []byte) ([]byte, []byte, error) {
	//log.Println("ExBUF: ", exbuf)
	tcpbuf := make([]byte, 65550)
	rLen := 0
	n, rErr := tcpCon.Stream.Read(tcpbuf)
	tcpbuf = tcpbuf[:n]
	buf := append(exbuf, tcpbuf...)
	rLen += n
	rLen += len(exbuf)
	//log.Println("Package Length Received: ", n)

	if rErr == nil {
		lth := buf[12:14]
		//log.Println(lth)
		length := binary.LittleEndian.Uint16(lth)
		//log.Println("Package Length Defined: ", length)
		for {
			if rLen < int(length)+14 {
				subbuf := make([]byte, 65550)
				n, rErr := tcpCon.Stream.Read(subbuf)
				subbuf = subbuf[:n]
				if rErr == nil {
					//log.Println("Package Length Received: ", n)
					buf = append(buf, subbuf...)
					rLen += n
				}
				continue
			} else if rLen > int(length)+14 {
				log.Println("RECV EXCESSIVE PACKAGE")
				break
			} else {
				log.Println("RECV PACKAGE COMPLETE")
				break
			}
		}
		//log.Println("BUF: ", buf)
		nonce, buf := buf[:12], buf[14:]
		data, buf := buf[:length], buf[length:]
		decData, err := GCMDecrypter(data, tcpCon.Passwd, nonce)
		if len(buf) == 0 {
			return decData, nil, err
		}
		return decData, buf, err
	}
	return nil, nil, rErr
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
	tcpCon.Stream.Write(finalPac)
}

func (tcpCon *TCPController) GetText() (string, error) {
	buf := make([]byte, 4096)
	_, rErr := tcpCon.Stream.Read(buf)

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

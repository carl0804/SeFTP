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
	Listener   net.Listener
	Passwd     [32]byte
}

func (tcpCon *TCPController) EstabListener() {
	ln, err := net.Listen("tcp", tcpCon.ServerAddr)
	if err != nil {
		log.Println(err)
	}
	tcpCon.Listener = ln
	log.Println("Listener Established.")
}

func (tcpCon *TCPController) CloseListener() {
	tcpCon.Listener.Close()
	log.Println("Listener closed.")
}

func (tcpCon *TCPController) SendByte(stream *smux.Stream, data []byte) {
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		log.Println(err.Error())
	}

	encByte := GCMEncrypter(data, tcpCon.Passwd, nonce)
	bs := make([]byte, 2)
	binary.LittleEndian.PutUint16(bs, uint16(len(encByte)))
	finalPac := append(append(nonce, bs...), encByte...)
	stream.Write(finalPac)
}

func (tcpCon *TCPController) GetByte(exbuf []byte, stream *smux.Stream) ([]byte, []byte, error) {
	//log.Println("ExBUF: ", exbuf)
	tcpbuf := make([]byte, 65550)
	rLen := 0
	n, rErr := stream.Read(tcpbuf)
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
				n, rErr := stream.Read(subbuf)
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

func (tcpCon *TCPController) SendText(stream *smux.Stream, text string) {
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		log.Println(err.Error())
	}

	encByte := GCMEncrypter([]byte(text), tcpCon.Passwd, nonce)
	bs := make([]byte, 2)
	binary.LittleEndian.PutUint16(bs, uint16(len(encByte)))
	finalPac := append(append(nonce, bs...), encByte...)
	stream.Write(finalPac)
}

func (tcpCon *TCPController) GetText(stream *smux.Stream) (string, error) {
	buf := make([]byte, 4096)
	_, rErr := stream.Read(buf)

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

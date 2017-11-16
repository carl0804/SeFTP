package Controller

import (
	"crypto/rand"
	"encoding/binary"
	"github.com/xtaci/kcp-go"
	"github.com/xtaci/smux"
	"io"
	"log"
	"net"
)

//Controller is the basis of two subcontrollers.
type Controller struct {
	ServerAddr string
	Passwd     [32]byte
}

//TraController is a mixed controller to control TCP and KCP.
type TraController interface {
	EstabListener()
	CloseListener()
	SendByte(*smux.Stream, []byte)
	GetByte([]byte, *smux.Stream) ([]byte, []byte, error)
	SendText(*smux.Stream, string)
	GetText(*smux.Stream) (string, error)
}

//SendByte is a function to send byte though socket.
func (Con *Controller) SendByte(stream *smux.Stream, data []byte) {
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		log.Println(err.Error())
	}

	encByte := GCMEncrypter(data, Con.Passwd, nonce)
	bs := make([]byte, 2)
	binary.LittleEndian.PutUint16(bs, uint16(len(encByte)))
	finalPac := append(append(nonce, bs...), encByte...)
	stream.Write(finalPac)
}

//GetByte is a function to get byte though socket.
func (Con *Controller) GetByte(exbuf []byte, stream *smux.Stream) ([]byte, []byte, error) {
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
		decData, err := GCMDecrypter(data, Con.Passwd, nonce)
		if len(buf) == 0 {
			return decData, nil, err
		}
		return decData, buf, err
	}
	return nil, nil, rErr
}

//SendText is a function to send text though socket.
func (Con *Controller) SendText(stream *smux.Stream, text string) {
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		log.Println(err.Error())
	}

	encByte := GCMEncrypter([]byte(text), Con.Passwd, nonce)
	bs := make([]byte, 2)
	binary.LittleEndian.PutUint16(bs, uint16(len(encByte)))
	finalPac := append(append(nonce, bs...), encByte...)
	stream.Write(finalPac)
}

//GetText is a function to get text though socket.
func (Con *Controller) GetText(stream *smux.Stream) (string, error) {
	buf := make([]byte, 4096)
	_, rErr := stream.Read(buf)

	if rErr == nil {
		nonce, buf := buf[:12], buf[12:]
		lth, buf := buf[:2], buf[2:]
		length := binary.LittleEndian.Uint16(lth)
		data, _ := buf[:length], buf[length:]
		decData, err := GCMDecrypter(data, Con.Passwd, nonce)
		return string(decData), err
	}
	return "", rErr
}

//TCPController is an interface to control a TCP Dial.
type TCPController struct {
	Controller
	Listener net.Listener
}

//EstabListener is a function to establish a listener.
func (tcpCon *TCPController) EstabListener() {
	ln, err := net.Listen("tcp", tcpCon.ServerAddr)
	if err != nil {
		log.Println(err)
	}
	tcpCon.Listener = ln
	log.Println("Listener Established.")
}

//CloseListener is a function to close TCP listener.
func (tcpCon *TCPController) CloseListener() {
	tcpCon.Listener.Close()
	log.Println("Listener closed.")
}

//KCPController is an interface to control a KCP Dial.
type KCPController struct {
	Controller
	Listener *kcp.Listener
}

//EstabListener is a function to establish a listener.
func (kcpCon *KCPController) EstabListener() {
	ln, err := kcp.ListenWithOptions(kcpCon.ServerAddr, nil, 10, 3)
	if err != nil {
		log.Println(err)
	}
	kcpCon.Listener = ln
	log.Println("Listener Established.")
}

//CloseListener is a function to close KCP connection.
func (kcpCon *KCPController) CloseListener() {
	kcpCon.Listener.Close()
	log.Println("Listener closed.")
}

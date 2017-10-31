package main

import (
	"fmt"
	"io"
	"net"
	//"encoding/hex"
	"./Controller"
	"flag"
	"log"
	"os"
	"strconv"
	"strings"
	//"bufio"
)

type Config struct {
	ServerAddr string
	Passwd     [32]byte
	ServerPort int
}

func (config *Config) Parse() {
	serverAddr := flag.String("s", "127.0.0.1", "Server IP Address")
	serverPort := flag.Int("p", 9080, "Server Port")
	plainPasswd := flag.String("k", "WELCOMETOTHEGRID", "Password")
	flag.Parse()

	var passwd [32]byte
	copy(passwd[:], *plainPasswd)

	config.ServerAddr = *serverAddr
	config.ServerPort = *serverPort
	config.Passwd = passwd
}

var SeFTPConfig = Config{}

func checkerr(e error) {
	if e != nil {
		log.Println(e)
	}
}

func GetOpenPort() int {
	laddr := net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0}
	listener, _ := net.ListenTCP("tcp4", &laddr)
	addr := listener.Addr()
	listener.Close()
	return addr.(*net.TCPAddr).Port
}

func handleCommand(seftpCon Controller.TCPController, conn net.Conn, plainCommand string) {
	command := strings.Fields(plainCommand)
	switch command[0] {
	case "GET":
		if _, err := os.Stat(string(command[1])); !os.IsNotExist(err) {
			subPort := GetOpenPort()
			subFtpCon := Controller.TCPController{ServerAddr: SeFTPConfig.ServerAddr + ":" + strconv.Itoa(subPort), Passwd: SeFTPConfig.Passwd}
			subFtpCon.EstabListener()
			defer subFtpCon.CloseListener()
			seftpCon.SendText(conn, "PASV PORT "+strconv.Itoa(subPort))
			for {
				// Get net.TCPConn object
				conn, err := subFtpCon.Listener.Accept()
				checkerr(err)
				plainEcho, err := subFtpCon.GetText(conn)
				checkerr(err)
				if plainEcho == "FILE SIZE" {
					f, err := os.Open(string(command[1]))
					checkerr(err)
					defer f.Close()
					fileInfo, err := f.Stat()
					fileSize := int(fileInfo.Size())
					subFtpCon.SendText(conn, "SIZE "+strconv.Itoa(fileSize))
					//result, err := subFtpCon.GetText(conn)
					//checkerr(err)
					//if result == "READY" {
					//	log.Println("CLIENT READY")
					sendSize := 0
					data := make([]byte, 60000)
					for sendSize < fileSize {
						result, err := subFtpCon.GetText(conn)
						checkerr(err)
						if result == "READY" {
							log.Println("CLIENT READY")
						} else if result == "REPEAT" {
							log.Println("CLIENT REQUEST PACKAGE RESENT")
							result, err := subFtpCon.GetText(conn)
							checkerr(err)
							if result == "READY" {
								subFtpCon.SendByte(conn, data)
								continue
							}
						}
						data := make([]byte, 60000)
						n, err := f.Read(data)
						if err != nil {
							if err == io.EOF {
								break
							}
							log.Println(err)
							return
						}
						data = data[:n]
						//log.Println("Data:", string(data))
						subFtpCon.SendByte(conn, data)
						sendSize += n
					}
					log.Println("FILE READ COMPLETE")
					result, err := subFtpCon.GetText(conn)
					checkerr(err)
					if result == "HALT" {
						log.Println("TRANSFER COMPLETE")
						break
					} else {
						log.Println("TRANSFER FAILED: ", result)
					}
				} else {
					subFtpCon.SendText(conn, "UNKNOWN COMMAND")
				}
			}
			log.Println("CLOSE SUBCONN")
			return
		} else {
			seftpCon.SendText(conn, "FILE NOT EXIST")
		}
		seftpCon.SendText(conn, "")
	case "CD":
		newPath := command[1]
		err := os.Chdir(newPath)
		checkerr(err)
		if err == nil {
			seftpCon.SendText(conn, "DIR CHANGED")
		} else {
			seftpCon.SendText(conn, "DIR CHANGE FAILED")
		}
	default:
		seftpCon.SendText(conn, "UNKNOWN COMMAND")
	}
}

func handleConnection(seftpCon Controller.TCPController, conn net.Conn) {
	log.Println("Handling new connection...")

	// Close connection when this function ends
	defer func() {
		log.Println("Closing connection...")
		conn.Close()
	}()

	for {
		text, rErr := seftpCon.GetText(conn)

		if rErr == nil {
			log.Println("Got Command:", text)
			handleCommand(seftpCon, conn, text)
			continue
		}

		if rErr == io.EOF {
			log.Println("END OF LINE.")

			break
		}

		fmt.Errorf(
			"Error while reading from",
			conn.RemoteAddr(),
			":",
			rErr,
		)
		break
	}
}

func main() {
	SeFTPConfig.Parse()
	seftpCon := Controller.TCPController{ServerAddr: SeFTPConfig.ServerAddr + ":" + strconv.Itoa(SeFTPConfig.ServerPort), Passwd: SeFTPConfig.Passwd}
	seftpCon.EstabListener()

	defer func() {
		seftpCon.CloseListener()
	}()

	for {
		// Get net.TCPConn object
		conn, err := seftpCon.Listener.Accept()
		checkerr(err)

		go handleConnection(seftpCon, conn)
	}
}

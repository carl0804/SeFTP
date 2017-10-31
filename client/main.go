package main

import (
	"./Controller"
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	//"io"
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

func main() {
	SeFTPConfig.Parse()
	seftpCon := Controller.TCPController{ServerAddr: SeFTPConfig.ServerAddr + ":" + strconv.Itoa(SeFTPConfig.ServerPort), Passwd: SeFTPConfig.Passwd}
	seftpCon.EstabConn()

	defer seftpCon.CloseConn()

	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter text: ")
		text, _ := reader.ReadString('\n')
		//log.Println(text)

		seftpCon.SendText(text)

		plainCommand, rErr := seftpCon.GetText()
		if rErr == nil {
			log.Println("Response From Server:", plainCommand)
			command := strings.Fields(plainCommand)
			switch command[0] {
			case "PASV":
				subftpCon := Controller.TCPController{ServerAddr: SeFTPConfig.ServerAddr + ":" + command[2], Passwd: SeFTPConfig.Passwd}
				subftpCon.EstabConn()
				defer subftpCon.CloseConn()
				subftpCon.SendText("FILE SIZE")
				plainCommand, err := subftpCon.GetText()
				checkerr(err)
				command := strings.Fields(plainCommand)
				switch command[0] {
				case "SIZE":
					fileSize, err := strconv.Atoi(command[1])
					checkerr(err)
					log.Println("FILE SIZE: ", fileSize)
					reader := bufio.NewReader(os.Stdin)
					fmt.Print("Enter File Name: ")
					fileName, _ := reader.ReadString('\n')
					f, err := os.Create(strings.Fields(fileName)[0])
					checkerr(err)
					defer f.Close()
					recvSize := 0
					for recvSize < fileSize {
						subftpCon.SendText("READY")
						buf, err := subftpCon.GetByte()
						if err != nil {
							log.Println("ERR: ", err.Error())
							subftpCon.SendText("REPEAT")
							continue
						}
						recvSize += len(buf)
						log.Println("RECV BYTE LENGTH: ", len(buf))
						f.Write(buf)
					}
					log.Println("FILE RECEIVED")
					subftpCon.SendText("HALT")
					continue
				}
			}
		}
	}
}

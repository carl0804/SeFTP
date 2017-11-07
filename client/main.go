package main

import (
	"./Controller"
	"encoding/binary"
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

var SeFTPConfig = Config{}

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
				if command[1] == "TCP" {
					subftpCon := Controller.TCPController{ServerAddr: SeFTPConfig.ServerAddr + ":" + command[2], Passwd: SeFTPConfig.Passwd}
					subftpCon.EstabConn()
					defer subftpCon.CloseConn()
					subftpCon.SendText("FILE SIZE")
					plainCommand, err := subftpCon.GetText()
					if !checkerr(err) {
						continue
					}
					command := strings.Fields(plainCommand)
					switch command[0] {
					case "SIZE":
						fileSize, err := strconv.Atoi(command[1])
						if !checkerr(err) {
							continue
						}
						log.Println("FILE SIZE: ", fileSize)
						reader := bufio.NewReader(os.Stdin)
						fmt.Print("Enter File Name: ")
						fileName, _ := reader.ReadString('\n')
						f, err := os.Create(strings.Fields(fileName)[0])
						if !checkerr(err) {
							continue
						}
						defer f.Close()
						recvSize := 0
						subftpCon.SendText("READY")
						var exbuf []byte
						var buf []byte
						for recvSize < fileSize {
							buf, exbuf, err = subftpCon.GetByte(exbuf)
							checkerr(err)
							recvSize += len(buf)
							//log.Println("RECV BYTE LENGTH: ", len(buf))
							f.Write(buf)
						}
						log.Println("FILE RECEIVED")
						subftpCon.SendText("HALT")
						continue
					}
				} else if command[1] == "UDP" {
					subftpCon := Controller.KCPController{ServerAddr: SeFTPConfig.ServerAddr + ":" + command[2], Passwd: SeFTPConfig.Passwd}
					subftpCon.EstabConn()
					defer subftpCon.CloseConn()
					subftpCon.SendText("FILE SIZE")
					plainCommand, err := subftpCon.GetText()
					if !checkerr(err) {
						continue
					}
					command := strings.Fields(plainCommand)
					switch command[0] {
					case "SIZE":
						fileSize, err := strconv.Atoi(command[1])
						if !checkerr(err) {
							continue
						}
						log.Println("FILE SIZE: ", fileSize)
						reader := bufio.NewReader(os.Stdin)
						fmt.Print("Enter File Name: ")
						fileName, _ := reader.ReadString('\n')
						f, err := os.Create(strings.Fields(fileName)[0])
						if !checkerr(err) {
							continue
						}
						defer f.Close()
						recvSize := 0
						subftpCon.SendText("READY")
						var exbuf []byte
						var buf []byte
						for recvSize+len(exbuf) < fileSize {
							buf, exbuf, err = subftpCon.GetByte(exbuf)
							checkerr(err)
							recvSize += len(buf)
							//log.Println("RECV BYTE LENGTH: ", len(buf))
							f.Write(buf)
						}
						if recvSize < fileSize {
							lth := exbuf[12:14]
							//log.Println(lth)
							length := binary.LittleEndian.Uint16(lth)
							nonce, exbuf := exbuf[:12], exbuf[14:]
							data, exbuf := exbuf[:length], exbuf[length:]
							decData, err := Controller.GCMDecrypter(data, SeFTPConfig.Passwd, nonce)
							checkerr(err)
							f.Write(decData)
						}
						log.Println("FILE RECEIVED")
						subftpCon.SendText("HALT")
						continue
					}
				}
			}
		}
	}
}

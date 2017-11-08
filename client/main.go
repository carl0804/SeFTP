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

func processRemoteCommand(plainClientCommand string, seftpCon Controller.TCPController) {
	clientCommand := strings.Fields(plainClientCommand)
	seftpCon.SendText(plainClientCommand)
	plainServerCommand, rErr := seftpCon.GetText()
	checkerr(rErr)
	serverCommand := strings.Fields(plainServerCommand)
	log.Println("Response From Server:", plainServerCommand)
	if clientCommand[0] == "GET" {
		if (len(clientCommand) <= 2) || (clientCommand[2] == "TCP") {
			subftpCon := Controller.TCPController{ServerAddr: SeFTPConfig.ServerAddr + ":" + serverCommand[2], Passwd: SeFTPConfig.Passwd}
			subftpCon.EstabConn()
			defer subftpCon.CloseConn()
			subftpCon.SendText("FILE SIZE")
			plainCommand, err := subftpCon.GetText()
			if !checkerr(err) {
				return
			}
			command := strings.Fields(plainCommand)
			switch command[0] {
			case "SIZE":
				fileSize, err := strconv.Atoi(command[1])
				if !checkerr(err) {
					return
				}
				log.Println("FILE SIZE: ", fileSize)
				reader := bufio.NewReader(os.Stdin)
				fmt.Print("Enter File Name: ")
				fileName, _ := reader.ReadString('\n')
				f, err := os.Create(strings.Fields(fileName)[0])
				if !checkerr(err) {
					return
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
				return
			}
		} else if clientCommand[2] == "UDP" {
			subftpCon := Controller.KCPController{ServerAddr: SeFTPConfig.ServerAddr + ":" + serverCommand[2], Passwd: SeFTPConfig.Passwd}
			subftpCon.EstabConn()
			defer subftpCon.CloseConn()
			subftpCon.SendText("FILE SIZE")
			plainCommand, err := subftpCon.GetText()
			if !checkerr(err) {
				return
			}
			command := strings.Fields(plainCommand)
			switch command[0] {
			case "SIZE":
				fileSize, err := strconv.Atoi(command[1])
				if !checkerr(err) {
					return
				}
				log.Println("FILE SIZE: ", fileSize)
				reader := bufio.NewReader(os.Stdin)
				fmt.Print("Enter File Name: ")
				fileName, _ := reader.ReadString('\n')
				f, err := os.Create(strings.Fields(fileName)[0])
				if !checkerr(err) {
					return
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
				return
			}
		}
	}
}

func processLocalCommand(plainClientCommand string) {
	clientCommand := strings.Fields(plainClientCommand)
	switch clientCommand[0] {
	case "cd":
		newPath := clientCommand[1]
		err := os.Chdir(newPath)
		if !checkerr(err) {
			log.Println("Dir change failed")
		} else {
			log.Println("Dir changed")
		}
	case "ls":
		var list []string
		if len(clientCommand) > 1 {
			path := clientCommand[1]
			list = Ls(path)
		} else {
			list = Ls("")
		}
		log.Println(strings.Join(list, " | "))
	case "exit":
		log.Println("Exit SeFTP")
		os.Exit(0)
	}
}

func processCommand(plainClientCommand string, seftpCon Controller.TCPController) {
	if IsUpper(strings.Fields(plainClientCommand)[0]) {
		processRemoteCommand(plainClientCommand, seftpCon)
	} else {
		processLocalCommand(plainClientCommand)
	}
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
		processCommand(text, seftpCon)
	}
}

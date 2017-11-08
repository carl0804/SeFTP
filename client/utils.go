package main

import (
	"golang.org/x/crypto/sha3"
	"flag"
	"log"
	"net"
	"strings"
	"io/ioutil"
	"bufio"
	"os"
	"fmt"
	"strconv"
	"./Controller"
	"encoding/binary"
	"gopkg.in/cheggaaa/pb.v2"
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

	passwd := GetSHA3Hash(*plainPasswd)

	config.ServerAddr = *serverAddr
	config.ServerPort = *serverPort
	config.Passwd = passwd
}

func GetSHA3Hash(text string) [32]byte {
	return sha3.Sum256([]byte(text))
}

func checkerr(e error) bool {
	if e != nil {
		log.Println(e)
		return false
	}
	return true
}

func GetOpenPort() (int, error) {
	laddr := net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0}
	listener, err := net.ListenTCP("tcp4", &laddr)
	if err == nil {
		addr := listener.Addr()
		listener.Close()
		return addr.(*net.TCPAddr).Port, nil
	}
	return 0, err
}

func IsUpper(str string) bool {
	return str == strings.ToUpper(str)
}

func Ls(path string) []string {
	if path == "" {
		path = "./"
	}
	files, err := ioutil.ReadDir(path)
	checkerr(err)

	var list []string

	for _, f := range files {
		list = append(list, f.Name())
	}
	return list
}

func GET(subftpInt interface{}) {
	if subftpCon, ok := subftpInt.(Controller.TCPController); ok {
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

			progressBar := pb.ProgressBarTemplate(`{{bar . | green}} {{speed . | blue }}`).Start(fileSize)
			defer progressBar.Finish()

			var exbuf []byte
			var buf []byte
			for recvSize < fileSize {
				buf, exbuf, err = subftpCon.GetByte(exbuf)
				checkerr(err)
				recvSize += len(buf)
				progressBar.Add(len(buf))
				//log.Println("RECV BYTE LENGTH: ", len(buf))
				f.Write(buf)
			}
			log.Println("FILE RECEIVED")
			subftpCon.SendText("HALT")
			return
		}
	} else if subftpCon, ok := subftpInt.(Controller.KCPController); ok {
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

			progressBar := pb.ProgressBarTemplate(`{{bar . | green}} {{speed . | blue }}`).Start(fileSize)
			defer progressBar.Finish()

			var exbuf []byte
			var buf []byte
			for recvSize+len(exbuf) < fileSize {
				buf, exbuf, err = subftpCon.GetByte(exbuf)
				checkerr(err)
				recvSize += len(buf)
				progressBar.Add(len(buf))
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

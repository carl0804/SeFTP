package main

import (
	"fmt"
	"strconv"
	"flag"
	"os"
	"bufio"
	"./Controller"
	"net"
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

func GetOpenPort() int {
	laddr := net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0}
	listener, _ := net.ListenTCP("tcp4", &laddr)
	addr := listener.Addr()
	listener.Close()
	return addr.(*net.TCPAddr).Port
}

func main() {
	SeFTPConfig.Parse()
	seftpCon := Controller.SeFTPController{ServerAddr: SeFTPConfig.ServerAddr + ":" + strconv.Itoa(SeFTPConfig.ServerPort), Passwd: SeFTPConfig.Passwd}
	seftpCon.EstabConn()

	defer func() {
		seftpCon.CloseConn()
	}()

	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter text: ")
		text, _ := reader.ReadString('\n')
		//fmt.Println(text)

		seftpCon.SendText(text)

		command, rErr := seftpCon.GetText()
		if rErr == nil {
			fmt.Println("Response From Server:", command)
		}
	}
}

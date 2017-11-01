package main

import (
	"log"
	"flag"
	"net"
	"golang.org/x/crypto/sha3"
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

package main

import (
	"log"
	"flag"
	"net"
	"golang.org/x/crypto/sha3"
	"strings"
	"io/ioutil"
	"os"
	"io"
	"encoding/hex"
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

func SHA3FileHash(filePath string) (result string, err error) {
	file, err := os.Open(filePath)
	if err != nil {
		return
	}
	defer file.Close()

	hash := sha3.New256()
	_, err = io.Copy(hash, file)
	if err != nil {
		return
	}

	result = hex.EncodeToString(hash.Sum(nil))
	return
}

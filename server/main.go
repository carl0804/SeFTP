package main

import (
	"fmt"
	"net"
	"io"
	//"encoding/hex"
	"flag"
	"strconv"
	"strings"
	"./Controller"
	"os"
	"log"
	"bufio"
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

func handleCommand(seftpCon Controller.SeFTPController, conn net.Conn, plainCommand string) {
	command := strings.Fields(plainCommand)
	switch command[0] {
	case "GET":
		if _, err := os.Stat(string(command[1])); !os.IsNotExist(err) {
			subPort := GetOpenPort()
			subFtpCon := Controller.SubFTPController{ServerAddr: SeFTPConfig.ServerAddr+":"+strconv.Itoa(subPort), Passwd:SeFTPConfig.Passwd}
			subFtpCon.EstabListener()
			seftpCon.SendText(conn, "PASV PORT "+strconv.Itoa(subPort))
			f, err := os.Open(string(command[1]))
			checkerr(err)
			fileReader := bufio.NewReader(f)
			fileBuffer, err := fileReader.Peek(1024)
			checkerr(err)
			log.Println("Got File: ", string(fileBuffer))
		} else {
			seftpCon.SendText(conn, "FILE NOT EXIST")
		}
		seftpCon.SendText(conn, "")
	default:
		seftpCon.SendText(conn, "UNKNOWN COMMAND")
	}
}

func handleConnection(seftpCon Controller.SeFTPController, conn net.Conn) {
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
	seftpCon := Controller.SeFTPController{ServerAddr: SeFTPConfig.ServerAddr + ":" + strconv.Itoa(SeFTPConfig.ServerPort), Passwd: SeFTPConfig.Passwd}
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

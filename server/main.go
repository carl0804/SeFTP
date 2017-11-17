package main

import (
	"./Controller"
	"io"
	//"encoding/hex"
	"log"
	"os"
	"strconv"
	"strings"
	//"bufio"
	"github.com/xtaci/smux"
)

//SeFTPConfig is a config predefined for convenience.
var SeFTPConfig = Config{}

func handleGet(seftpCon Controller.TraController, stream *smux.Stream, command []string) {
	if _, err := os.Stat(string(command[1])); !os.IsNotExist(err) {
		subPort, err := GetOpenPort()
		if !checkerr(err) {
			return
		}
		fileName := string(command[1])
		if (len(command) <= 2) || (command[2] == "TCP") {
			subFtpCon := &Controller.TCPController{
				Controller: Controller.Controller{
					ServerAddr: SeFTPConfig.ServerAddr + ":" + strconv.Itoa(subPort),
					Passwd:     SeFTPConfig.Passwd,
				},
			}
			subFtpCon.EstabListener()
			defer subFtpCon.CloseListener()

			seftpCon.SendText(stream, "PASV TCP "+strconv.Itoa(subPort))

			subconn, _ := subFtpCon.Listener.Accept()
			session, _ := smux.Server(subconn, nil)

			// Accept a stream
			substream, _ := session.AcceptStream()
			GET(substream, seftpCon, fileName)
		} else if command[2] == "UDP" {
			subFtpCon := &Controller.KCPController{
				Controller: Controller.Controller{
					ServerAddr: ":" + strconv.Itoa(subPort),
					Passwd:     SeFTPConfig.Passwd,
				},
			}
			subFtpCon.EstabListener()
			defer subFtpCon.CloseListener()

			seftpCon.SendText(stream, "PASV UDP "+strconv.Itoa(subPort))
			subconn, err := subFtpCon.Listener.AcceptKCP()
			if !checkerr(err) {
				return
			}
			subconn.SetStreamMode(true)
			subconn.SetWriteDelay(true)
			subconn.SetNoDelay(1, 20, 2, 1)
			subconn.SetWindowSize(1024, 1024)
			subconn.SetMtu(1350)
			// Setup server side of smux
			session, err := smux.Server(subconn, nil)
			if !checkerr(err) {
				return
			}

			// Accept a stream
			substream, err := session.AcceptStream()
			GET(substream, seftpCon, fileName)
		}
	} else {
		seftpCon.SendText(stream, "FILE NOT EXIST")
	}
	seftpCon.SendText(stream, "")
}

func handlePost(seftpCon Controller.TraController, stream *smux.Stream, command []string) {
	subPort, err := GetOpenPort()
	if !checkerr(err) {
		return
	}
	filePath := string(command[1])
	if (len(command) <= 2) || (command[2] == "TCP") {
		subFtpCon := &Controller.TCPController{
			Controller: Controller.Controller{
				ServerAddr: SeFTPConfig.ServerAddr + ":" + strconv.Itoa(subPort),
				Passwd:     SeFTPConfig.Passwd,
			},
		}
		subFtpCon.EstabListener()
		defer subFtpCon.CloseListener()

		seftpCon.SendText(stream, "PASV TCP "+strconv.Itoa(subPort))
		// Get net.TCPConn object
		subconn, err := subFtpCon.Listener.Accept()
		if !checkerr(err) {
			return
		}
		// Setup server side of smux
		session, err := smux.Server(subconn, nil)
		if !checkerr(err) {
			return
		}

		// Accept a stream
		substream, err := session.AcceptStream()
		if !checkerr(err) {
			return
		}
		log.Println("ACCEPT SUBSTREAM")
		POST(substream, subFtpCon, filePath)
	} else if command[2] == "UDP" {
		subFtpCon := &Controller.KCPController{
			Controller: Controller.Controller{
				ServerAddr: ":" + strconv.Itoa(subPort),
				Passwd:     SeFTPConfig.Passwd,
			},
		}
		subFtpCon.EstabListener()
		defer subFtpCon.CloseListener()

		seftpCon.SendText(stream, "PASV UDP "+strconv.Itoa(subPort))
		// Get net.TCPConn object
		subconn, err := subFtpCon.Listener.AcceptKCP()
		if !checkerr(err) {
			return
		}
		subconn.SetStreamMode(true)
		subconn.SetWriteDelay(true)
		subconn.SetNoDelay(1, 20, 2, 1)
		subconn.SetWindowSize(1024, 1024)
		subconn.SetMtu(1350)
		// Setup server side of smux
		session, err := smux.Server(subconn, nil)
		if !checkerr(err) {
			return
		}

		// Accept a stream
		substream, err := session.AcceptStream()
		if !checkerr(err) {
			return
		}
		log.Println("ACCEPT SUBSTREAM")
		POST(substream, subFtpCon, filePath)
	}
}

func handleCommand(seftpCon Controller.TraController, stream *smux.Stream, plainCommand string) {
	command := strings.Fields(plainCommand)
	switch command[0] {
	case "GET":
		handleGet(seftpCon, stream, command)
	case "POST":
		handlePost(seftpCon, stream, command)

	case "CD":
		newPath := command[1]
		err := os.Chdir(newPath)
		if !checkerr(err) {
			seftpCon.SendText(stream, "DIR CHANGE FAILED")
		} else {
			seftpCon.SendText(stream, "DIR CHANGED")
		}
	case "LS":
		var list []string
		if len(command) > 1 {
			path := command[1]
			list = Ls(path)
		} else {
			list = Ls("")
		}
		seftpCon.SendText(stream, strings.Join(list, " | "))
	case "RM":
		if len(command) > 1 {
			err := os.Remove(command[1])
			if !checkerr(err) {
				seftpCon.SendText(stream, "RM FAILED")
			} else {
				seftpCon.SendText(stream, "RM SUCCEEDED")
			}
		} else {
			log.Println("NO SPECIFIC FILE")
		}
	case "SHA3SUM":
		if len(command) > 1 {
			sum, err := SHA3FileHash(command[1])
			checkerr(err)
			seftpCon.SendText(stream, sum)
		} else {
			seftpCon.SendText(stream, "No specific file")
		}
	default:
		seftpCon.SendText(stream, "UNKNOWN COMMAND")
	}
}

func handleConnection(seftpCon Controller.TraController, stream *smux.Stream) {
	log.Println("Handling new connection...")

	// Close connection when this function ends
	defer func() {
		log.Println("Closing connection...")
		stream.Close()
	}()

	for {
		text, rErr := seftpCon.GetText(stream)

		if rErr == nil {
			log.Println("Got Command:", text)
			handleCommand(seftpCon, stream, text)
			continue
		}

		if rErr == io.EOF {
			log.Println("END OF LINE.")

			break
		}
		break
	}
}

func main() {
	SeFTPConfig.Parse()
	seftpCon := &Controller.TCPController{
		Controller: Controller.Controller{
			ServerAddr: SeFTPConfig.ServerAddr + ":" + strconv.Itoa(SeFTPConfig.ServerPort),
			Passwd:     SeFTPConfig.Passwd,
		},
	}
	seftpCon.EstabListener()

	defer seftpCon.CloseListener()

	for {
		// Get net.TCPConn object
		conn, err := seftpCon.Listener.Accept()
		if !checkerr(err) {
			continue
		}
		// Setup server side of smux
		session, err := smux.Server(conn, nil)
		if !checkerr(err) {
			continue
		}

		// Accept a stream
		stream, err := session.AcceptStream()
		if !checkerr(err) {
			continue
		}

		go handleConnection(seftpCon, stream)
	}
}

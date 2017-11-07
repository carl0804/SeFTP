package main

import (
	"./Controller"
	"fmt"
	"io"
	//"encoding/hex"
	"log"
	"os"
	"strconv"
	"strings"
	//"bufio"
	"github.com/xtaci/smux"
	"time"
)

var SeFTPConfig = Config{}

func handleCommand(seftpCon Controller.TCPController, stream *smux.Stream, plainCommand string) {
	command := strings.Fields(plainCommand)
	switch command[0] {
	case "GET":
		if _, err := os.Stat(string(command[1])); !os.IsNotExist(err) {
			subPort, err := GetOpenPort()
			if !checkerr(err) {
				return
			}
			subFtpCon := Controller.TCPController{ServerAddr: SeFTPConfig.ServerAddr + ":" + strconv.Itoa(subPort), Passwd: SeFTPConfig.Passwd}
			subFtpCon.EstabListener()
			defer subFtpCon.CloseListener()
			seftpCon.SendText(stream, "PASV PORT "+strconv.Itoa(subPort))
			for {
				// Get net.TCPConn object
				subconn, err := subFtpCon.Listener.Accept()
				if !checkerr(err) {
					continue
				}
				// Setup server side of smux
				session, err := smux.Server(subconn, nil)
				if !checkerr(err) {
					continue
				}

				// Accept a stream
				substream, err := session.AcceptStream()
				if !checkerr(err) {
					continue
				}
				plainEcho, err := subFtpCon.GetText(substream)
				if !checkerr(err) {
					continue
				}
				if plainEcho == "FILE SIZE" {
					f, err := os.Open(string(command[1]))
					if !checkerr(err) {
						continue
					}
					defer f.Close()
					fileInfo, err := f.Stat()
					if !checkerr(err) {
						continue
					}
					fileSize := int(fileInfo.Size())
					subFtpCon.SendText(substream, "SIZE "+strconv.Itoa(fileSize))
					//result, err := subFtpCon.GetText(conn)
					//checkerr(err)
					//if result == "READY" {
					//	log.Println("CLIENT READY")
					sendSize := 0
					result, err := subFtpCon.GetText(substream)
					if !checkerr(err) {
						continue
					}
					if result == "READY" {
						log.Println("CLIENT READY")
						for sendSize < fileSize {
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
							subFtpCon.SendByte(substream, data)
							sendSize += n
							time.Sleep(time.Microsecond)
						}
					}
					log.Println("FILE READ COMPLETE")
					result, err = subFtpCon.GetText(substream)
					if !checkerr(err) {
						break
					}
					if result == "HALT" {
						log.Println("TRANSFER COMPLETE")
						break
					} else {
						log.Println("TRANSFER FAILED: ", result)
					}
				} else {
					subFtpCon.SendText(substream, "UNKNOWN COMMAND")
				}
			}
			log.Println("CLOSE SUBCONN")
			return
		} else {
			seftpCon.SendText(stream, "FILE NOT EXIST")
		}
		seftpCon.SendText(stream, "")
	case "CD":
		newPath := command[1]
		err := os.Chdir(newPath)
		if !checkerr(err) {
			seftpCon.SendText(stream, "DIR CHANGE FAILED")
		} else {
			seftpCon.SendText(stream, "DIR CHANGED")
		}
	default:
		seftpCon.SendText(stream, "UNKNOWN COMMAND")
	}
}

func handleConnection(seftpCon Controller.TCPController, stream *smux.Stream) {
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

		fmt.Errorf(
			"Error while reading from",
			stream.RemoteAddr(),
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

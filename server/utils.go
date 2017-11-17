package main

import (
	"./Controller"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"github.com/xtaci/smux"
	"golang.org/x/crypto/sha3"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

//Config is the config struct for SeFTP.
type Config struct {
	ServerAddr string
	Passwd     [32]byte
	ServerPort int
}

//Parse is a function to parse flag config to Config struct.
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

//GetSHA3Hash is a function to get SHA3 hash of a string.
func GetSHA3Hash(text string) [32]byte {
	return sha3.Sum256([]byte(text))
}

//checkerr is a function to check if there is error.
func checkerr(e error) bool {
	if e != nil {
		log.Println(e)
		return false
	}
	return true
}

//GetOpenPort is a function to get an open TCP port.
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

//IsUpper is a function to determine if string uses uppercase.
func IsUpper(str string) bool {
	return str == strings.ToUpper(str)
}

//Ls is a function to handle LS request.
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

//SHA3FileHash is a function to get file's SHA3 hash.
func SHA3FileHash(filePath string) (result string, err error) {
	file, err := os.Open(filePath)
	if err != nil {
		return
	} //Get is a function to handle GET request.

	defer file.Close()

	hash := sha3.New256()
	_, err = io.Copy(hash, file)
	if err != nil {
		return
	}

	result = hex.EncodeToString(hash.Sum(nil))
	return
}

func GET(substream *smux.Stream, subFtpCon Controller.TraController, fileName string) {
	plainEcho, err := subFtpCon.GetText(substream)
	if !checkerr(err) {
		return
	}
	if plainEcho == "FILE SIZE" {
		f, err := os.Open(fileName)
		if !checkerr(err) {
			return
		}
		defer f.Close()
		fileInfo, err := f.Stat()
		if !checkerr(err) {
			return
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
			return
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
			return
		}
		if result == "HALT" {
			log.Println("TRANSFER COMPLETE")
			return
		} else {
			log.Println("TRANSFER FAILED: ", result)
		}
	} else {
		subFtpCon.SendText(substream, "UNKNOWN COMMAND")
	}
	log.Println("CLOSE SUBCONN")
	return
}

func POST(substream *smux.Stream, subFtpCon Controller.TraController, filePath string) {
	plainEcho, err := subFtpCon.GetText(substream)
	if !checkerr(err) {
		return
	}
	echo := strings.Fields(plainEcho)
	log.Println("ECHO: ", plainEcho)
	if (echo[0] != "SIZE") || (len(echo) != 2) {
		return
	}
	fileSize, err := strconv.Atoi(echo[1])
	if !checkerr(err) {
		return
	}
	f, err := os.Create(strings.Fields(filePath)[0])
	if !checkerr(err) {
		return
	}
	defer f.Close()
	recvSize := 0
	subFtpCon.SendText(substream, "READY")

	var exbuf []byte
	var buf []byte
	for recvSize+len(exbuf) < fileSize {
		buf, exbuf, err = subFtpCon.GetByte(exbuf, substream)
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
		data, _ := exbuf[:length], exbuf[length:]
		decData, err := Controller.GCMDecrypter(data, SeFTPConfig.Passwd, nonce)
		checkerr(err)
		f.Write(decData)
	}
	log.Println("FILE RECEIVED")
	subFtpCon.SendText(substream, "HALT")
	time.Sleep(time.Second)
	return
}

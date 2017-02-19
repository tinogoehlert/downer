package xdcc

import (
	"encoding/binary"
	"log"
	"net"
	"os"
	"strconv"
	"strings"

	irc "github.com/thoj/go-ircevent"
)

const (
	RequestStatusRequested = 1
	RequestStatusQueued    = 2
	RequestStatusActive    = 3
	RequestStatusDone      = 4
)

type Request struct {
	Query  string
	File   string
	Status int
}

type Server struct {
	Name       string
	connection *irc.Connection
}

var serverList map[string]*Server

// Connect connects to a IRC Server
func Connect(server string, nick string, user string) *Server {

	ircobj := irc.IRC(nick, user)
	ircobj.Connect(server)

	connection := Server{server, ircobj}
	if len(serverList) == 0 {
		serverList = make(map[string]*Server)
	}
	serverList[server] = &connection
	ircobj.AddCallback("PRIVMSG", func(event *irc.Event) {
		ParseMessage(&connection, event.Message(), event.Nick, strings.ToLower(event.Arguments[0]))
	})

	ircobj.AddCallback("CTCP", func(event *irc.Event) {
		parts := strings.Split(event.Message(), " ")
		if len(parts) > 4 && parts[1] == "SEND" {
			log.Println(parts)
			ipnum, _ := strconv.ParseUint(parts[3], 10, 32)
			port, _ := strconv.ParseUint(parts[4], 10, 32)
			if port > 0 {
				go startDccDownload(uint32(ipnum), uint32(port), parts[2])
			} else {
				startPassiveDccDownload(int(ipnum), int(port), parts[2])
			}
		}
	})

	return &connection
}

func startDccDownload(ip uint32, port uint32, filename string) error {
	senderIP := make(net.IP, 4)
	binary.BigEndian.PutUint32(senderIP, ip)
	strAddr := senderIP.String() + ":" + strconv.FormatUint(uint64(port), 10)
	log.Printf("dcc connect to %s\n", strAddr)

	conn, err := net.Dial("tcp", strAddr)

	if err != nil {
		log.Println(err)
		return err
	}

	defer conn.Close()

	f, err := os.Create(filename)
	if err != nil {
		log.Println(err)
		return err
	}
	defer f.Close()
	// 3 MB Buffer
	b := make([]byte, 3145728)
	for {
		lenght, rerr := conn.Read(b)
		if rerr != nil {
			log.Println(rerr)
			return rerr
		}
		if lenght > 0 {
			f.Write(b[:lenght])
		}
	}
}

func startPassiveDccDownload(ip int, port int, filename string) {
	//net.IPv4(byte(ip&0xFF), byte((ip>>8)&0xFF), byte((ip>>16)&0xFF), byte((ip>>24)&0xFF))
	log.Println("passive dcc! -.-")
}

func RequestPackage(query string) *Request {
	pathq := strings.Split(query, "/")

	conn := serverList[pathq[0]]
	log.Printf("%s => xdcc send #%s\n", pathq[2], pathq[3])
	conn.connection.Privmsgf(pathq[2], "XDCC SEND #%s", pathq[3])
	return &Request{
		Status: RequestStatusRequested,
		Query:  query}
}

// Connected determines if you're still connected to IRC server
func (xc Server) Connected() bool {
	return xc.connection.Connected()
}

// Join to a channel on this Server
func (xc Server) Join(channel string) {
	xc.connection.Join(channel)
}

// Disconnect to a channel on this Server
func (xc Server) Disconnect() {
	xc.connection.Disconnect()

}

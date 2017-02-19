package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
	"tinogoehlert/downer/xdcc"

	"github.com/BurntSushi/toml"
)

type config struct {
	Database string
	Xdcc     map[string]xdccserver
}

type xdccserver struct {
	Server   string
	Username string
	Nickname string
	Channels []string
}

func apiErrorRequest(reason string, w http.ResponseWriter) {
	w.WriteHeader(http.StatusBadRequest)

	ret, _ := json.Marshal(struct {
		Status string `json:"status"`
		Reason string `json:"reason"`
	}{"error", reason})
	log.Println(string(ret))
	w.Write(ret)
	return
}

func apiXdccPackSearch(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	bs, _ := ioutil.ReadAll(r.Body)
	w.Header().Set("Content-Type", "application/json")
	var ps *struct {
		Filter string `json:"filter"`
	}

	if err := json.Unmarshal(bs, &ps); err != nil {
		apiErrorRequest(err.Error(), w)
		return
	}

	retcount := 0
	DbXdccSearchPackets(ps.Filter, func(found []*XdccPackageRecord) {
		res, _ := json.Marshal(found)
		retcount = len(found)
		w.Write(res)
	})
	elapsed := time.Since(start)
	log.Printf("xdcc search %s returned %d packages and took %s", ps.Filter, retcount, elapsed)
}

func apiXdccDownload(w http.ResponseWriter, r *http.Request) {

	bs, _ := ioutil.ReadAll(r.Body)
	var dlreq struct {
		Path string `json:"path"`
	}

	if err := json.Unmarshal(bs, &dlreq); err != nil {
		apiErrorRequest(err.Error(), w)
	}

	xr := xdcc.RequestPackage(dlreq.Path)
	DbXdccAddRequest(dlreq.Path, xr)
}

func main() {

	var conf config
	if _, err := toml.DecodeFile("config.toml", &conf); err != nil {
		log.Fatal(err)
		os.Exit(-2)
	}

	DbOpenBolt(conf.Database)
	defer DbCloseBolt()

	for _, xdccserver := range conf.Xdcc {
		xdccCon := xdcc.Connect(xdccserver.Server,
			xdccserver.Nickname,
			xdccserver.Username)
		defer xdccCon.Disconnect()
		for _, channel := range xdccserver.Channels {
			log.Println("join ", channel)
			xdccCon.Join(channel)
		}
	}

	http.HandleFunc("/xdcc/packages/search", apiXdccPackSearch)
	http.HandleFunc("/xdcc/download", apiXdccDownload)
	http.HandleFunc("/sys/config", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "GET" {
			res, _ := json.Marshal(conf)
			w.Write(res)
		} else {
			w.WriteHeader(http.StatusNotImplemented)
		}
	})

	xdcc.OnPackage(func(server *xdcc.Server, nick string, channel string, pack *xdcc.Package) {
		DbXdccAddPackage(server.Name, nick, channel, pack)
	})

	xdcc.OnOffer(func(server *xdcc.Server, bot *xdcc.Bot) {
		DbXdccAddBot(server.Name, bot)
	})

	http.ListenAndServe("localhost:8080", nil)
	done := make(chan bool, 1)
	<-done
}

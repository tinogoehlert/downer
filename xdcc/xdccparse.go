package xdcc

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"time"
)

// Bot representation of a Bot
type Bot struct {
	Nick         string
	Channel      string
	PackageCount int
	SlotsOpen    int
	SlotsMax     int
	RecordStr    string
	Lastseen     int64
}

// Package representation of a Package
type Package struct {
	Name     string
	SizeStr  string
	Number   int
	bot      Bot
	Lastseen int64
}

// JSONBot get Package from JSON
func JSONBot(data []byte) (*Bot, error) {
	var p *Bot
	err := json.Unmarshal(data, &p)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// JSONPackage get Package from JSON
func JSONPackage(data []byte) (*Package, error) {
	var p *Package
	err := json.Unmarshal(data, &p)
	if err != nil {
		return nil, err
	}
	return p, nil
}

var xdccPackageCallback = func(server *Server, nick string, channel string, pack *Package) {
	fmt.Printf("[PACK] (%s) %s\n", pack.SizeStr, pack.Name)
}

var xdccOfferCallback = func(server *Server, bot *Bot) {
	fmt.Printf("[OFFER] %s - speed: %s\tslots: %d\n", bot.Nick, bot.RecordStr, bot.SlotsOpen)
}

var regexXdccpack = regexp.MustCompile(`#(\d+).+?\d+x \[ *(<?\d+.*?)\] +(.*)$`)
var regexXdccoffer = regexp.MustCompile(`\*\* ((?P<packs>\d{1,3}) packs)|((Min: (?P<speed_min>\d{1,5}.\d{1,2}[KkMm][bB])\/s))|((?P<slots_open>\d{1,3}) of (?P<slots_count>\d{1,3}) slots open)|([Q|q]ueue: (?P<queue_current>\d{1,2})\/(?P<queue_count>\d{1,2}))|(Max: (?P<speed_max>\d{1,5}.\d{1,2}[KkMm][bB])\/s)|(Record: (?P<speed_record>\d{1,5}.\d{1,2}[KkMm][bB])\/s)|`)
var regexXdccband = regexp.MustCompile(`\*\* Bandwidth Usage \*\* Current: (\d{1,5}.\d{1,2}[KkMm][bB])\/s`)
var regexRemoveColor = regexp.MustCompile("[\u0002-\u000F]")

func scan(str string, rg *regexp.Regexp) (bool, []string) {
	match := rg.FindStringSubmatch(str)
	return (len(match) > 0), match
}

func scanNamed(str string, rg *regexp.Regexp) (bool, map[string]string) {
	result := make(map[string]string)
	for _, match := range rg.FindAllStringSubmatch(str, -1) {
		for i, name := range rg.SubexpNames() {
			// result[name] = match[i]
			if i != 0 && match[i] != "" && name != "" {
				result[name] = match[i]
			}
		}
	}

	return (len(result) > 0), result
}

// OnPackage calls a function when an Offer was fetched
func OnPackage(pkg func(server *Server, nick string, channel string, pack *Package)) {
	xdccPackageCallback = pkg
}

// OnOffer calls a function when an Offer was fetched
func OnOffer(offer func(server *Server, bot *Bot)) {
	xdccOfferCallback = offer
}

// ParseMessage tries to identify XDCC related Messages within a string.
func ParseMessage(server *Server, msg string, nick string, channel string) {
	msg = regexRemoveColor.ReplaceAllString(msg, "")

	if ok, pack := scan(msg, regexXdccpack); ok {
		packageNumber, _ := strconv.ParseInt(pack[1], 10, 32)
		xdccPackageCallback(server, nick, channel, &Package{
			Number:   int(packageNumber),
			SizeStr:  pack[2],
			Name:     pack[3],
			Lastseen: time.Now().Unix()})
	} else if ok, bandwidth := scan(msg, regexXdccband); ok {
		fmt.Printf("[BAND] %s\n", bandwidth[1])
	} else if ok, offer := scanNamed(msg, regexXdccoffer); ok {
		slotsopen, _ := strconv.ParseInt(offer["slots_open"], 10, 32)
		slotsmax, _ := strconv.ParseInt(offer["slots_max"], 10, 32)
		packcount, _ := strconv.ParseInt(offer["packs"], 10, 32)

		xdccOfferCallback(server, &Bot{
			PackageCount: int(packcount),
			Nick:         nick,
			Channel:      channel,
			SlotsOpen:    int(slotsopen),
			SlotsMax:     int(slotsmax),
			RecordStr:    offer["speed_record"],
			Lastseen:     time.Now().Unix()})
	}
}

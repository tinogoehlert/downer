package main

import (
	"bytes"
	"encoding/json"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"tinogoehlert/downer/xdcc"

	"github.com/boltdb/bolt"
)

type XdccPackageRecord struct {
	Path    string        `json:"path"`
	Package *xdcc.Package `json:"package"`
}

var boltdb *bolt.DB

func DbOpenBolt(path string) {
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		log.Fatal(err)
		os.Exit(2)
	} else {
		boltdb = db
	}
}

func DbCloseBolt() {
	boltdb.Close()
}

func getNickBucket(tx *bolt.Tx, server string, channel string, nick string) (*bolt.Bucket, error) {
	rootbucket, _ := tx.CreateBucketIfNotExists([]byte("xdcc"))
	sbucket, err := rootbucket.CreateBucketIfNotExists([]byte(server))
	if err != nil {
		return nil, err
	}

	chanbucket, _ := sbucket.CreateBucketIfNotExists([]byte(channel))
	botbucket, _ := chanbucket.CreateBucketIfNotExists([]byte(nick))
	return botbucket, nil
}

func DbXdccAddBot(server string, bot *xdcc.Bot) error {
	boltdb.Update(func(tx *bolt.Tx) error {
		tx.CreateBucketIfNotExists([]byte("Xdcc"))
		bucket, err := tx.Bucket([]byte("Xdcc")).CreateBucketIfNotExists([]byte("offers"))

		if err != nil {
			log.Fatal(err)
			return err
		}

		serialized, _ := json.Marshal(bot)
		bucket.Put([]byte(server+"/"+bot.Channel+"/"+bot.Nick), []byte(serialized))
		return nil
	})
	return nil
}

func DbXdccAddPackage(server string, nick string, channel string, pack *xdcc.Package) error {
	boltdb.Update(func(tx *bolt.Tx) error {
		tx.CreateBucketIfNotExists([]byte("Xdcc"))
		bucket, err := tx.Bucket([]byte("Xdcc")).CreateBucketIfNotExists([]byte("packages"))

		if err != nil {
			log.Fatal(err)
			return err
		}

		serialized, _ := json.Marshal(pack)
		bucket.Put([]byte(server+"/"+channel+"/"+nick+"/"+strconv.Itoa(pack.Number)), serialized)
		return nil
	})
	return nil
}

func DbXdccAddRequest(query string, r *xdcc.Request) error {
	boltdb.Update(func(tx *bolt.Tx) error {
		tx.CreateBucketIfNotExists([]byte("Xdcc"))
		bucket, _ := tx.Bucket([]byte("Xdcc")).CreateBucketIfNotExists([]byte("requests"))
		serialized, _ := json.Marshal(r)
		if err := bucket.Put([]byte(query), []byte(serialized)); err != nil {
			log.Println(err.Error())
		}
		return nil
	})
	return nil
}

func DbXdccSearchRequest(query string) (*xdcc.Request, error) {
	var r *xdcc.Request
	err := boltdb.Update(func(tx *bolt.Tx) error {
		buck, _ := tx.Bucket([]byte("Xdcc")).CreateBucketIfNotExists([]byte("requests"))
		c := buck.Cursor()
		for k, v := c.Seek([]byte(query)); k != nil && bytes.HasPrefix(k, []byte(query)); k, v = c.Next() {
			json.Unmarshal(v, &r)
		}
		return nil
	})

	return r, err
}

func DbXdccSearchPackets(pattern string, matches func([]*XdccPackageRecord)) {

	boltdb.View(func(tx *bolt.Tx) error {
		buck := tx.Bucket([]byte("Xdcc")).Bucket([]byte("packages"))
		words := strings.Split(strings.ToLower(pattern), " ")
		wordcount := len(words)
		repattern := "(?i)"

		for _, word := range words {
			repattern += "(" + word + ")" + "|"
		}

		repattern = strings.TrimRight(repattern, "|")

		re := regexp.MustCompile(repattern)
		found := make([]*XdccPackageRecord, 0)
		c := buck.Cursor()
		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			matches := re.FindAllString(string(v), -1)
			if len(matches) == wordcount {
				jp, _ := xdcc.JSONPackage(v)
				found = append(found, &XdccPackageRecord{
					Path:    string(k),
					Package: jp})
			}
		}
		matches(found)
		return nil
	})
}

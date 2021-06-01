package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/flikas/dht-recorder/dht"
	_ "github.com/huin/goupnp"
	"log"
	"net/http"
	_ "net/http/pprof"
	"time"
)

type file struct {
	Path   []interface{} `json:"path"`
	Length int           `json:"length"`
}

type bitTorrent struct {
	InfoHash string `json:"infohash"`
	Name     string `json:"name"`
	Files    []file `json:"files,omitempty"`
	Length   int    `json:"length,omitempty"`
}

func main() {
	go func() {
		err := http.ListenAndServe(":6060", nil)
		if err != nil {
			return
		}
	}()

	w := dht.NewWire(65536, 1024, 256)
	go func() {
		for resp := range w.Response() {
			//fmt.Println(resp)

			metadata, err := dht.Decode(resp.MetadataInfo)
			if err != nil {
				continue
			}
			info := metadata.(map[string]interface{})

			//fmt.Println(info)

			if _, ok := info["name"]; !ok {
				continue
			}

			bt := bitTorrent{
				InfoHash: hex.EncodeToString(resp.InfoHash),
				Name:     info["name"].(string),
			}

			if v, ok := info["files"]; ok {
				files := v.([]interface{})
				bt.Files = make([]file, len(files))

				for i, item := range files {
					f := item.(map[string]interface{})
					bt.Files[i] = file{
						Path:   f["path"].([]interface{}),
						Length: f["length"].(int),
					}
				}
			} else if _, ok := info["length"]; ok {
				bt.Length = info["length"].(int)
			}

			data, err := json.Marshal(bt)
			if err == nil {
				fmt.Printf("%s\n\n", data)
			}
		}
	}()
	go w.Run()

	config := dht.NewCrawlConfig()
	config.MaxNodes = 500000
	config.OnAnnouncePeer = func(infoHash, ip string, port int) {
		w.Request([]byte(infoHash), ip, port)
	}
	config.OnGetPeersResponse = func(infoHash string, peer *dht.Peer) {
		fmt.Printf("GOT PEER: <%s:%d>\n", peer.IP, peer.Port)
	}
	d := dht.New(config)

	go func() {
		for {
			// ubuntu-14.04.2-desktop-amd64.iso
			err := d.GetPeers("546cf15f724d19c4319cc17b179d7e035f89c1f4")
			if err != nil && err != dht.ErrNotReady {
				log.Fatal(err)
			}

			if err == dht.ErrNotReady {
				time.Sleep(time.Second * 1)
				continue
			}

			break
		}
	}()

	fmt.Println("Started on 6060/http, 6881/udp")
	d.Run()
}

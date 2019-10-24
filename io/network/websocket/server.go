package websocket

import (
	baseConnect "HelloWorld/io/network/connect"
	"HelloWorld/io/network/websocket/connect"
	"flag"
	"gorilla/websocket"
	"log"
	"net/http"
	"time"
)

var server *http.Server
var upGrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var i uint32 = 0

func Server(address string) {

	var addr = flag.String("addr", address, "http service address")

	flag.Parse()
	log.SetFlags(0)

	mux := http.NewServeMux()
	server = &http.Server{
		Addr:         *addr,
		WriteTimeout: time.Second * 4,
		Handler:      mux,
	}
	mux.HandleFunc("/ws", Connect)
	log.Fatal(server.ListenAndServe())
}

func Connect(w http.ResponseWriter, r *http.Request) {
	conn, err := upGrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	i++
	// 写入本地连接列表
	connector := &connect.Connector{Conn: conn, ID: i}
	baseConnect.Add(connector)
	go connector.Connected()
}

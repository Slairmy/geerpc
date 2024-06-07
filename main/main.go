package main

import (
	"encoding/json"
	"fmt"
	"geerpc"
	"geerpc/codec"
	"log"
	"net"
	"time"
)

func startServer(addr chan string) {
	l, err := net.Listen("tcp", ":8000")
	if err != nil {
		log.Fatal("network error: ", err)
	}
	log.Println("start rpc server on", l.Addr())
	addr <- l.Addr().String()

	// rpc服务监听
	geerpc.Accept(l)
}

func main() {
	addr := make(chan string)
	// 开启rpc服务
	go startServer(addr)

	// 建立tcp连接
	conn, _ := net.Dial("tcp", <-addr)
	defer func() { _ = conn.Close() }()

	time.Sleep(time.Second)
	_ = json.NewEncoder(conn).Encode(geerpc.DefaultOption)
	c := codec.NewGobCodec(conn)
	for i := 0; i < 5; i++ {
		h := &codec.Header{
			ServiceMethod: "foo.sum",
			Seq:           uint64(i),
		}

		// 往tcp连接中写入header头
		_ = c.Write(h, fmt.Sprintf("geerpc req %d", h.Seq))
		_ = c.ReadHeader(h)
		var reply string
		_ = c.ReadBody(&reply)
		log.Println("reply: ", reply)
	}
}

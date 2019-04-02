package main

import (
	"fmt"
	"log"
	"net"

	"gitee.com/rocket049/rpc2d"
)

type Server rpc2d.ProviderType

var count = 0

func (self *Server) Show(arg string, reply *int) error {
	fmt.Printf("Recv: %s, count: %d\n", arg, count)
	*reply = count
	count++
	var ret int
	self.Client.Call("Client.Show", fmt.Sprintf("callback:%s.", arg), &ret)
	return nil
}

func main() {
	l, err := net.Listen("tcp", "127.0.0.1:5678")
	if err != nil {
		log.Fatal("Listen:", err)
	}
	defer l.Close()
	p := new(Server)
	node1, err := rpc2d.Accept(l, p)
	if err != nil {
		log.Fatal("Accept:", err)
	}
	defer node1.Close()
	p.Client = node1.Client
	var s string
	var ret int
	for i := 0; i < 5; i++ {
		s = fmt.Sprintf("server message %d\n", i)
		node1.Client.Call("Client.Show", s, &ret)
		fmt.Printf("Return:%d\n", ret)
	}

	select {}
}

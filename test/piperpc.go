package main

import (
	"fmt"
	"net"
	"net/rpc"
)

type My int

var count = 10

func (self *My) Show(arg string, reply *int) error {
	fmt.Printf("Recv: %s\n", arg)
	*reply = count
	count++
	return nil
}

func main() {
	p1, p2 := net.Pipe()
	server := rpc.NewServer()
	server.Register(new(My))
	go server.ServeConn(p1)
	client := rpc.NewClient(p2)
	var s string
	var ret int
	for i := 0; i < 5; i++ {
		fmt.Scanln(&s)
		client.Call("My.Show", s, &ret)
		fmt.Println(ret)
	}
}

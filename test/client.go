package main

import (
	"fmt"
	"log"

	"gitee.com/rocket049/rpc2d"
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
	p := new(My)
	node1 := rpc2d.NewRpcNode(p)
	err := node1.Dial("127.0.0.1:5678")
	if err != nil {
		log.Fatal("Dial:", err)
	}
	//p.Client = node1.Client
	defer node1.Close()
	var s string
	var ret int
	for i := 0; i < 5; i++ {
		fmt.Scanln(&s)
		node1.Client.Call("My.Show", s, &ret)
		fmt.Printf("Return: %d\n", ret)
	}
}

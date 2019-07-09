//测试和 gitee.com/rocket049/pipeconn 的兼容性
package main

import (
	"fmt"

	"gitee.com/rocket049/rpc2d"

	"gitee.com/rocket049/pipeconn"
)

type Client int

var count = 10

func (self *Client) Show(arg string, reply *int) error {
	fmt.Printf("Recv: %s\n", arg)
	*reply = count
	count++
	return nil
}

func main() {
	p := new(Client)
	conn, err := pipeconn.NewClientPipeConn("./pipeserver")
	if err != nil {
		return
	}
	node1 := rpc2d.NewRpcNodeByConn(p, conn)
	defer node1.Close()
	var s string
	var ret int
	for i := 0; i < 5; i++ {
		s = fmt.Sprintf("client message %d\n", i)
		node1.Client.Call("Server.Show", s, &ret)
		fmt.Printf("Return: %d\n", ret)
	}
	select {}
}

//测试和 gitee.com/rocket049/pipeconn 的兼容性
package main

import (
	"fmt"

	"gitee.com/rocket049/pipeconn"
	"gitee.com/rocket049/rpc2d"
)

type Args struct {
	A, B int
}

type Quotient struct {
	Quo, Rem int
}

type Server rpc2d.ProviderType

var count = 0

func (self *Server) Show(arg string, reply *int) error {
	*reply = count
	count++
	var ret int
	self.Client.Call("Client.Show", fmt.Sprintf("callback:%s.", arg), &ret)
	return nil
}

func main() {
	p := new(Server)
	conn := pipeconn.NewServerPipeConn()
	node1 := rpc2d.NewRpcNodeByConn(p, conn)
	defer node1.Close()
	p.Client = node1.Client
	var s string
	var ret int
	for i := 0; i < 5; i++ {
		s = fmt.Sprintf("server message %d\n", i)
		node1.Client.Call("Client.Show", s, &ret)
	}

	select {}
}

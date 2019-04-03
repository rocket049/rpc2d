package rpc2d

import (
	"fmt"
	"log"
	"net"
	"os"
	"testing"
)

type Client int

var count = 10

func (self *Client) Show(arg string, reply *int) error {
	//fmt.Printf("Recv: %s\n", arg)
	*reply = count
	count++
	return nil
}

func BenchmarkClient(b *testing.B) {
	b.Log("start benchmark")
	testProc(b.N)
}

func TestClient(t *testing.T) {
	t.Log("start test")
	err := testProc(1000)
	if err != nil {
		t.Fatal(err)
	}
}

func TestMain(m *testing.M) {
	ch := make(chan int, 1)
	go testServer(ch)
	<-ch
	os.Exit(m.Run())
}

func testProc(n int) error {
	p := new(Client)
	node1 := NewRpcNode(p)
	err := node1.Dial("127.0.0.1:5678")
	if err != nil {
		return err
	}
	defer node1.Close()
	var s string
	var ret int
	for i := 0; i < n; i++ {
		s = fmt.Sprintf("client message %d\n", i)
		err := node1.Client.Call("Server.Show", s, &ret)
		if err != nil {
			return err
		}
	}
	return nil
}

type Server ProviderType

var sCount = 0

func (self *Server) Show(arg string, reply *int) error {
	//fmt.Printf("Recv: %s, count: %d\n", arg, count)
	*reply = sCount
	sCount++
	var ret int
	self.Client.Call("Client.Show", fmt.Sprintf("callback:%s.", arg), &ret)
	return nil
}
func testServer(ch chan int) {
	l, err := net.Listen("tcp", "127.0.0.1:5678")
	if err != nil {
		log.Fatal("Listen:", err)
	}
	defer l.Close()
	ch <- 1
	for {
		p := new(Server)
		node1, err := Accept(l, p)
		if err != nil {
			log.Fatal("Accept:", err)
		}
		p.Client = node1.Client
		defer node1.Close()
	}
}

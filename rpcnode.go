//rpc2d 双向 RPC 调用，可以实现从服务器 CALLBACK 客户端 API，基于 "net/rpc" 原生库
package rpc2d

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
	"log"
	"net"
	"net/rpc"
	"sync"
)

//wrap message( []byte ): "T uint8 + length uint16 + bytes [length]byte".  T = S/C/E
const (
	S = byte('S') //Flag : Server Message
	C = byte('C') //Flag : Client Message
	E = byte('E') //Flag : Error
)

//Pool: bytes.Buffer, use : bufPool.Get().(*bytes.Buffer)
var bufPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

func newBuffer() *bytes.Buffer {
	return bufPool.Get().(*bytes.Buffer)
}

//Provider Type is NOT must fit this. But this struct can help CALLBACK. See test server.go/client.go
type ProviderType struct {
	Client *rpc.Client
	Data   interface{}
}

//RpcNode double direction RPC
type RpcNode struct {
	Server         *rpc.Server
	Client         *rpc.Client
	connC1, connC2 net.Conn
	connS1, connS2 net.Conn
	remote         net.Conn
}

//NewRpcNode create new Rpc.Node ,init rpc.Server with service provider
func NewRpcNode(provider interface{}) *RpcNode {
	res := new(RpcNode)
	res.Server = rpc.NewServer()
	res.Server.Register(provider)
	return res
}

//WrapSend wrap and split message, send to remote
func (self *RpcNode) wrapSend(t byte, msg []byte, conn io.Writer) (nbytes int, e error) {
	len1 := len(msg)
	n := len1 / 65535
	m := uint16(len1 % 65535)
	//log.Printf("length:%d  split:%d  last:%d\n", len1, n, m)
	var h1 = [3]byte{t, 0xff, 0xff}
	//h1[0] = t
	//binary.BigEndian.PutUint16(h1[1:2], 65535)
	bufconn := bufio.NewWriter(conn)
	for i := 0; i < n; i++ {
		//send
		p := msg[i*65535 : i*65535+65535]
		b := newBuffer()
		b.Write(h1[:])
		b.Write(p)
		_, e := bufconn.Write(b.Bytes())
		if e != nil {
			return 0, e
		}
		bufPool.Put(b)
	}
	if m > 0 {
		//send
		binary.BigEndian.PutUint16(h1[1:3], m)
		p := msg[n*65535 : n*65535+int(m)]
		b := newBuffer()
		b.Reset()
		b.Write(h1[:])
		b.Write(p)
		_, e := bufconn.Write(b.Bytes())
		if e != nil {
			return 0, e
		}
		//log.Printf("length:%d  split:%d  last:%d\nfrom %c:%v\n", len1, n, m, t, b.Bytes())
		bufPool.Put(b)
	}
	err := bufconn.Flush()
	if err != nil {
		log.Printf("WrapSend:%v\n", err)
		return 0, err
	} else {
		return len1, nil
	}
}

//wrapRecv receive message from remote. Next: route to server or client
func (self *RpcNode) wrapRecv(conn io.Reader) (msg []byte, t byte) {
	//bufconn := bufio.NewReader(conn)
	var h1 [3]byte
	n, _ := io.ReadFull(conn, h1[:])
	if n != 3 {
		return nil, E
	}
	length := binary.BigEndian.Uint16(h1[1:])
	buf1 := make([]byte, int(length))
	n, _ = io.ReadFull(conn, buf1)
	if n == int(length) {
		return buf1, h1[0]
	} else {
		return nil, E
	}
}

//proxyLoop proxy between remote and local server/client,redirect/wrapsend messages
func (self *RpcNode) proxyLoop(conn net.Conn) {
	self.connS1, self.connS2 = net.Pipe()
	self.connC1, self.connC2 = net.Pipe()
	//self.Server = rpc.NewServer()
	go func() {
		self.Server.ServeConn(self.connS1)
		//log.Println("end serve")
	}()
	self.Client = rpc.NewClient(self.connC1)
	self.remote = conn
	//loop next
	go self.localToRemote(self.connC2, C)
	go self.localToRemote(self.connS2, S)
	go self.remoteToLocal() //block
}

func (self *RpcNode) remoteToLocal() {
LOOP1:
	for {
		msg, t := self.wrapRecv(self.remote)
		switch t {
		case S:
			self.connC2.Write(msg)
			//log.Printf("to C:%v\n", msg)
		case C:
			self.connS2.Write(msg)
			//log.Printf("to S:%v\n", msg)
		case E:
			break LOOP1
		}
	}
	self.remote.Close()
	log.Println("remote disconnect")
}

func (self *RpcNode) localToRemote(from io.ReadCloser, t byte) {
	var buf = make([]byte, 512)
	for {
		n, err := from.Read(buf)
		if n > 0 {
			_, err := self.wrapSend(t, buf[:n], self.remote)
			if err != nil {
				log.Printf("WrapSend:%v\n", err)
				break
			}
		} else {
			log.Printf("local Read:%v\n", err)
			break
		}
	}
	from.Close()
	log.Printf("local disconnect: %c\n", t)
}

//Dial connect to remote, and link local server/client,use after NewRpcNode
func (self *RpcNode) Dial(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	self.proxyLoop(conn)
	return nil
}

//Close close
func (self *RpcNode) Close() {
	self.Client.Close()
	self.connC2.Close()
	self.connC1.Close()
	self.connS2.Close()
	self.connS1.Close()
}

//Accept accept remote connection,and link local server/client
func Accept(l net.Listener, provider interface{}) (*RpcNode, error) {
	conn, err := l.Accept()
	if err != nil {
		return nil, err
	}
	node1 := NewRpcNode(provider)
	node1.proxyLoop(conn)
	return node1, nil
}

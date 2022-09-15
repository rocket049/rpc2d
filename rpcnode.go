package rpc2d

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"net/rpc/jsonrpc"
	"sync"
	"time"
)

// Can connect to RPC service using HTTP CONNECT to rpcPath.
var connected = "200 Connected to TOPVAS Go RPC"

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

//RpcNode double direction RPC
type RpcNode struct {
	Server         *rpc.Server
	Client         *rpc.Client
	connC1, connC2 net.Conn
	connS1, connS2 net.Conn
	remote         net.Conn
	ErrFunc        []func(*RpcNode, error)
}

//NewRpcNode create new Rpc.Node ,init rpc.Server with service provider
func NewRpcNode(name string, rcvr any) *RpcNode {
	res := new(RpcNode)
	res.Server = rpc.NewServer()
	res.Server.RegisterName(name, rcvr)

	return res
}

//WrapSend wrap and split message, send to remote
func (self *RpcNode) wrapSend(t byte, msg []byte, conn io.Writer) (nbytes int, e error) {
	len1 := len(msg)
	n := len1 / 65535
	m := uint16(len1 % 65535)
	var h1 = [3]byte{t, 0xff, 0xff}
	bufConn := bufio.NewWriter(conn)
	b := newBuffer()
	for i := 0; i < n; i++ {
		//send
		p := msg[i*65535 : i*65535+65535]
		b.Reset()
		b.Write(h1[:])
		b.Write(p)
		_, e := bufConn.Write(b.Bytes())
		if e != nil {
			return 0, e
		}
	}
	if m > 0 {
		//send
		binary.BigEndian.PutUint16(h1[1:3], m)
		p := msg[n*65535 : n*65535+int(m)]
		b.Reset()
		b.Write(h1[:])
		b.Write(p)
		_, e := bufConn.Write(b.Bytes())
		if e != nil {
			return 0, e
		}
	}
	bufPool.Put(b)
	err := bufConn.Flush()
	if err != nil {
		log.Printf("WrapSend:%v\n", err)
		return 0, err
	} else {
		return len1, nil
	}
}

//wrapRecv receive message from remote. Next: route to server or client
func (self *RpcNode) wrapRecv(conn io.Reader) (msg []byte, t byte) {
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
	go func() {
		self.Server.ServeCodec(jsonrpc.NewServerCodec(self.connS1))
	}()
	self.Client = jsonrpc.NewClient(self.connC1)
	self.remote = conn
	//loop next
	go self.localToRemote(self.connC2, C)
	go self.localToRemote(self.connS2, S)
	go self.remoteToLocal() //block
}

func (self *RpcNode) remoteToLocal() {
	var errCountS, errCountC int
	var bufRemote = bufio.NewReader(self.remote)

	for {
		msg, t := self.wrapRecv(bufRemote)
		switch t {
		case S:
			errCountS = 0
			self.connC2.Write(msg)
		case C:
			errCountC = 0
			self.connS2.Write(msg)
		case E:
			errCountS++
			errCountC++
			break
		}
		if errCountC > 30 || errCountS > 30 {
			break
		}
	}
	self.remote.Close()

	err := errors.New("wrapRecv error")
	go self.notifyException(err)
}

func (self *RpcNode) localToRemote(from io.ReadCloser, t byte) {
	var n int
	var err error
	var buf = make([]byte, 512)
	for {
		n, err = from.Read(buf)
		if n > 0 {
			_, err = self.wrapSend(t, buf[:n], self.remote)
			if err != nil {
				break
			}
		} else {
			break
		}
	}
	from.Close()
	go self.notifyException(err)
}

//Dial connect to remote, and link local server/client,use after NewRpcNode
// @network		network name   tcp/tpc6/udp/udp6... dial.parseNetwork()
// @address     host and port net.JoinHostPort()
// @path     	http rpc url path
// @isTLS       to support tls such https set this is true
// @timeout		dial time out time
func (self *RpcNode) Dial(network, address string, path string, isTLS bool, timeout time.Duration) error {

	var err error
	var conn net.Conn

	netDialer := &net.Dialer{
		Timeout:   20 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

	if isTLS {
		conn, err = tls.DialWithDialer(netDialer, network, address, tlsConfig)
	} else {
		conn, err = net.DialTimeout(network, address, netDialer.Timeout)
	}

	if err != nil {
		return err
	}

	io.WriteString(conn, "CONNECT "+path+" HTTP/1.0\n\n")

	// Require successful HTTP response
	// before switching to RPC protocol.
	resp, err := http.ReadResponse(bufio.NewReader(conn), &http.Request{Method: "CONNECT"})
	if err == nil && resp.Status == connected {
		self.proxyLoop(conn)
		return nil
	}
	if err == nil {
		err = errors.New("unexpected HTTP response: " + resp.Status)
	}
	conn.Close()
	return &net.OpError{
		Op:   "dial-http",
		Net:  network + " " + address,
		Addr: nil,
		Err:  err,
	}
}

//Close to close all
func (self *RpcNode) Close() {
	self.Client.Close()
	self.connC2.Close()
	self.connC1.Close()
	self.connS2.Close()
	self.connS1.Close()
	self.remote.Close()
}

//AcceptConn accept remote connection,and link local server/client
// @conn 	http Hijack
// @name	rpc register name
// @rcvr	rpc receiver's concrete type
func AcceptConn(conn net.Conn, name string, rcvr interface{}) (*RpcNode, error) {
	node1 := NewRpcNode(name, rcvr)
	node1.proxyLoop(conn)
	return node1, nil
}

// notifyException call ErrFun when exception come in.
func (self *RpcNode) notifyException(e error) {
	for _, f := range self.ErrFunc {
		try(f, self, e)
	}
}

// Try tries to run a function and recovers from a panic
func try(f func(s *RpcNode, e error), r *RpcNode, e error) {
	defer func() { recover() }()
	f(r, e)
}


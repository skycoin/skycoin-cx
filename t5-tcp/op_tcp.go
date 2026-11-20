package tcp

import (
	"context"
	"log"
	"net"
	"net/rpc"
	"time"

	"github.com/skycoin/skycoin-cx/t0-types" // thin: only Write_str; no depth gain
	"github.com/skycoin/skycoin-cx/t3-ast"
)

// todo need to find a way to support Listener and connection interface
// VIOLATION: state outside State. Owner: t5-tcp; moves to TCPListenerMng in phase 1.
var ln net.Listener

var lc net.ListenConfig

// VIOLATION: state outside State. Owner: t5-tcp; moves to TCPConnMng in phase 1.
var conn net.Conn

// VIOLATION: init chaining. Init 3: the tcp package RPC server, called from Go
// package init.
// VIOLATION: state outside State. Owner: t5-tcp; moves to RPCServerMng in phase 1.
var DefaultServer = rpc.NewServer()

func opTCPDial(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	var network, address, errorstring *ast.CXArgument
	if inputs[0].TypeSignature.Type == ast.TYPE_CXARGUMENT_DEPRECATE {
		network = prgrm.GetCXArgFromArray(ast.CXArgumentIndex(inputs[0].TypeSignature.Meta))
	} else {
		panic("type is not type cx argument deprecate\n\n")
	}

	if inputs[1].TypeSignature.Type == ast.TYPE_CXARGUMENT_DEPRECATE {
		address = prgrm.GetCXArgFromArray(ast.CXArgumentIndex(inputs[1].TypeSignature.Meta))
	} else {
		panic("type is not type cx argument deprecate\n\n")
	}

	if outputs[0].TypeSignature.Type == ast.TYPE_CXARGUMENT_DEPRECATE {
		errorstring = prgrm.GetCXArgFromArray(ast.CXArgumentIndex(outputs[0].TypeSignature.Meta))
	} else {
		panic("type is not type cx argument deprecate\n\n")
	}

	fp := inputs[0].FramePointer

	log.Println("network", network)
	log.Println("address", address)

	var err error
	conn, err = net.Dial("tcp", "localhost:9000")

	if err != nil {
		types.Write_str(prgrm, prgrm.Memory, ast.GetFinalOffset(prgrm, fp, errorstring, nil), err.Error())
	}

	conn.Close()
}

func opTCPClose(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	ln.Close()
}

func opTCPAccept(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	conn, _ = ln.Accept()
	conn.Close()
}

func opTCPListen(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	var network, address, errorstring *ast.CXArgument
	if inputs[0].TypeSignature.Type == ast.TYPE_CXARGUMENT_DEPRECATE {
		network = prgrm.GetCXArgFromArray(ast.CXArgumentIndex(inputs[0].TypeSignature.Meta))
	} else {
		panic("type is not type cx argument deprecate\n\n")
	}

	if inputs[1].TypeSignature.Type == ast.TYPE_CXARGUMENT_DEPRECATE {
		address = prgrm.GetCXArgFromArray(ast.CXArgumentIndex(inputs[1].TypeSignature.Meta))
	} else {
		panic("type is not type cx argument deprecate\n\n")
	}

	if outputs[0].TypeSignature.Type == ast.TYPE_CXARGUMENT_DEPRECATE {
		errorstring = prgrm.GetCXArgFromArray(ast.CXArgumentIndex(outputs[0].TypeSignature.Meta))
	} else {
		panic("type is not type cx argument deprecate\n\n")
	}

	fp := inputs[0].FramePointer

	log.Println("network", network)
	log.Println("address", address)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)

	defer cancel()

	var err error

	ln, err = lc.Listen(ctx, "tcp", ":9000")

	ln.Close()

	if err != nil {
		types.Write_str(prgrm, prgrm.Memory, ast.GetFinalOffset(prgrm, fp, errorstring, nil), err.Error())
	}
}

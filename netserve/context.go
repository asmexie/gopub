package netserve

import (
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/asmexie/gopub/common"
	"github.com/muroachanf/go-logger/logger"
	gcc "github.com/patrickmn/go-cache"
)

// NetContext ...
type NetContext struct {
	logVerbose bool
	peerAdrr   string
	aeskey     []byte
	recviv     []byte
	sendiv     []byte
	sendsig    uint64
	recvsig    uint64
	seq        uint32
	ack        uint32
	state      byte
	nonce      uint64
	updateiv   bool
	stream     bool
	size       int
	packsize   int
	ackSetChan chan uint32
}

var (
	gTunnelTasks   chan *common.TunnelTask = make(chan *common.TunnelTask, 1)
	gUdpNetContext *gcc.Cache              = gcc.New(time.Minute*5, time.Minute*5)
)

func init() {
	go common.RunTunnelTasks(gTunnelTasks)
}

func NewNetContext(peerAddr string) *NetContext {
	return &NetContext{peerAdrr: peerAddr, ackSetChan: make(chan uint32, 1)}
}

func GetUdpNetContext(peerAddr string) (ctx *NetContext) {
	common.TunnelExec(gTunnelTasks, func() {
		v, ok := gUdpNetContext.Get(peerAddr)
		if !ok || v == nil {
			ctx = NewNetContext(peerAddr)
			gUdpNetContext.Set(peerAddr, ctx, gcc.DefaultExpiration)

		} else {
			ctx = v.(*NetContext)
		}
	})
	return
}

func (context *NetContext) checkSetAck(ack uint32) (ok bool) {
	context.Verbosef("set context %v ack %v", context.peerAdrr, ack)
	context.ackSetChan <- ack
	ok = context.ack == 0 || context.ack != ack
	context.ack = <-context.ackSetChan
	return
}

func (context *NetContext) Verbose(s string) {
	if context.logVerbose {
		logger.DebugN(1, s)
	}
}

func (context *NetContext) Verbosef(format string, v ...interface{}) {
	if context.logVerbose {
		logger.DebugN(1, fmt.Sprintf(format, v...))
	}
}

func (context *NetContext) BuildAckHdr(hdr *TransPacketHdr) {
	if context.state == 2 {
		hdr.Msgtype = 2
	} else {
		if context.stream {
			hdr.Msgtype = 4
		} else {
			hdr.Msgtype = 3
		}
	}

	hdr.Version = 2
	hdr.Seq = context.seq
	hdr.Nonce = uint64(time.Now().Unix())
	context.seq = hdr.Seq
	context.nonce = hdr.Nonce
	context.UpdateIv()
}

func (context *NetContext) GetPackSize(n, blockSize int) (int, int) {
	datasize := context.size - n
	if datasize > context.packsize {
		datasize = context.packsize
	}
	pksize := RoundUp(datasize, blockSize)
	if pksize == datasize {
		pksize += blockSize
	}
	return datasize, pksize
}

func (context *NetContext) CalcStreamSize(blockSize int) int {
	n := 0
	size := 0

	for n < context.size {
		datasize, packsize := context.GetPackSize(n, blockSize)
		size += packsize
		n += datasize
	}
	return size
}

func (context *NetContext) UpdateIv() {
	if !context.updateiv {
		return
	}
	context.Verbosef("updating iv:\n recviv % x\n nonce:%x\n seq:%x\n recvchecksum:%x",
		context.recviv, context.nonce, context.seq, context.recvsig)
	h := md5.New()
	h.Write(context.recviv)
	binary.Write(h, binary.LittleEndian, context.nonce)
	binary.Write(h, binary.LittleEndian, context.seq)
	binary.Write(h, binary.LittleEndian, context.recvsig)
	context.sendiv = h.Sum(nil)
	context.updateiv = false
}

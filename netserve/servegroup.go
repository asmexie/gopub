package netserve

import (
	"bufio"
	"context"
	"io"
	"sync"

	"bitbucket.org/muroachanf/gopub/common"
)

var (
	bufioReaderPool sync.Pool
	bufioWriterPool sync.Pool
)

func newBufioReader(r io.Reader) *bufio.Reader {
	if v := bufioReaderPool.Get(); v != nil {
		br := v.(*bufio.Reader)
		br.Reset(r)
		return br
	}
	// Note: if this reader size is every changed, update
	// TestHandlerBodyClose's assumptions.
	return bufio.NewReader(r)
}

func putBufioReader(br *bufio.Reader) {
	br.Reset(nil)
	bufioReaderPool.Put(br)
}

func newBufioWriter(w io.Writer, size int) *bufio.Writer {
	if v := bufioWriterPool.Get(); v != nil {
		bw := v.(*bufio.Writer)
		bw.Reset(w)
		return bw
	}

	return bufio.NewWriterSize(w, size)
}

func putBufioWriter(bw *bufio.Writer) {
	bw.Reset(nil)
	bufioWriterPool.Put(bw)
}

// ServeGroup ...
type ServeGroup struct {
	terminate bool
	nsc       NetServeConfig
	cipher    TransCipher
	d         PDecoder
	hd        APIHandler
}

// ListenAndServeServeGroups ...
func ListenAndServeServeGroups(ctx context.Context, netconfigs []NetServeConfig, f NameToAPIHandler) {
	for _, nsc := range netconfigs {
		hd := f(nsc.HandlerName)
		tcp := &ServeGroup{
			nsc:    nsc,
			cipher: NewTransCipher(nsc.Cipher),
			d:      newDecoder(nsc, hd),
			hd:     hd,
		}
		go tcp.Serve(ctx)
	}
}

// Serve ...
func (sg *ServeGroup) Serve(ctx context.Context) {
	defer func() {
		if x := recover(); x != nil {
			common.LogError(x)
		}
	}()
	sg.terminate = false
	tp := sg.nsc
	var serves []NetServe
	for _, nettype := range tp.NetType {
		for _, ip := range tp.ListenIP {
			for _, port := range tp.Port {
				serves = append(serves, newNetServe(sg, nettype, ip, port))
			}
		}
	}

	for _, l := range serves {
		go l.Serve(ctx)
	}
}

// Stop ...
func (sg *ServeGroup) Stop() {
	sg.terminate = true
}

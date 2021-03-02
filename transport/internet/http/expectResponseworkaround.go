package http

import (
	"context"
	"io"
)

type expectResponseWorkaroundReader struct {
	io.Reader
	firstRead bool
}

func (e *expectResponseWorkaroundReader) Read(p []byte) (n int, err error) {
	if e.firstRead == true {
		var discard [1]byte
		nw, errw := e.Reader.Read(discard[:])
		if errw != nil {
			return 0, err
		}
		if nw != 1 {
			return 0, newError("Unable to discard content as expectResponseWorkaroundReader")
		}
		e.firstRead = false
	}
	return e.Reader.Read(p)
}

func newExpectResponseWorkaroundReader(reader io.Reader) io.Reader {
	return &expectResponseWorkaroundReader{
		Reader:    reader,
		firstRead: true,
	}
}

type ReadCloserPromise struct {
	io.ReadCloser
	resolve    context.Context
	cancelFunc context.CancelFunc
	err        error
}

func (e *ReadCloserPromise) Close() (err error) {
	<-e.resolve.Done()
	if e.err != nil {
		return newError("reader promise rejected").Base(err)
	}
	return e.ReadCloser.Close()
}

func (e *ReadCloserPromise) Read(p []byte) (n int, err error) {
	<-e.resolve.Done()
	if e.err != nil {
		return 0, newError("reader promise rejected").Base(err)
	}
	return e.ReadCloser.Read(p)
}

func (e *ReadCloserPromise) Resolve(reader io.ReadCloser) {
	e.ReadCloser = reader
	e.cancelFunc()
}

func (e *ReadCloserPromise) Reject(err error) {
	e.err = err
	e.cancelFunc()
}

func NewReadCloserPromise() *ReadCloserPromise {
	resolveCtx, resolveCancel := context.WithCancel(context.Background())
	return &ReadCloserPromise{
		resolve:    resolveCtx,
		cancelFunc: resolveCancel,
	}
}

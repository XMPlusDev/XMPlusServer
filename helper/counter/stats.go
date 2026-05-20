package counter

import (
    "fmt"
    "sync"
	
	"github.com/xtls/xray-core/common"
	"github.com/xtls/xray-core/common/buf"
)

type DeltaFunc func(upload, download int64) bool

type StatWriter struct {
	Writer  buf.Writer
	Storage *TrafficStorage
	Delta   DeltaFunc
	Cancel  func()
	Email   string
	once    sync.Once
}

func (w *StatWriter) WriteMultiBuffer(mb buf.MultiBuffer) error {
	n := int64(mb.Len())
	err := w.Writer.WriteMultiBuffer(mb)
	if n > 0 {
		w.Storage.DownCounter.Add(n)
		if w.Delta != nil && w.Delta(0, n) {
			w.once.Do(func() {
				if w.Cancel != nil {
					w.Cancel()
				}
			})
			return fmt.Errorf("traffic quota exceeded for %s", w.Email)
		}
	}
	return err
}

func (w *StatWriter) Close() error { return common.Close(w.Writer) }
func (w *StatWriter) Interrupt()   { common.Interrupt(w.Writer) }

type StatReader struct {
	Reader  buf.TimeoutReader
	Storage *TrafficStorage
	Delta   DeltaFunc
	Cancel  func()
	Email   string
	once    sync.Once
}

func (r *StatReader) ReadMultiBuffer() (buf.MultiBuffer, error) {
	mb, err := r.Reader.ReadMultiBuffer()
	if err != nil {
		return nil, err
	}
	if mb.Len() > 0 {
		n := int64(mb.Len())
		r.Storage.UpCounter.Add(n)
		if r.Delta != nil && r.Delta(n, 0) {
			r.once.Do(func() {
				if r.Cancel != nil {
					r.Cancel()
				}
			})
			buf.ReleaseMulti(mb)
			return nil, fmt.Errorf("traffic quota exceeded for %s", r.Email)
		}
	}
	return mb, nil
}

func (r *StatReader) ReadMultiBufferTimeout(d interface{ Nanoseconds() int64 }) (buf.MultiBuffer, error) {
	return r.ReadMultiBuffer()
}

func (r *StatReader) Interrupt() { common.Interrupt(r.Reader) }
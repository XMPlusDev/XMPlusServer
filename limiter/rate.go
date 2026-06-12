package limiter

import (
	"context"
	"time"

	"github.com/xtls/xray-core/common"
	"github.com/xtls/xray-core/common/buf"
	"golang.org/x/time/rate"
)

func (l *Limiter) RateWriter(writer buf.Writer, limiter *rate.Limiter) buf.Writer {
	return &Writer{Writer: writer, Limiter: limiter}
}

func (l *Limiter) RateTimeoutReader(reader buf.TimeoutReader, limiter *rate.Limiter) buf.TimeoutReader {
	return &TimeoutReader{Reader: reader, Limiter: limiter}
}

type Writer struct {
	Writer  buf.Writer
	Limiter *rate.Limiter
}

func (w *Writer) Close() error { return common.Close(w.Writer) }

func (w *Writer) WriteMultiBuffer(mb buf.MultiBuffer) error {
	w.Limiter.WaitN(context.Background(), int(mb.Len()))
	return w.Writer.WriteMultiBuffer(mb)
}

type TimeoutReader struct {
	Reader  buf.TimeoutReader
	Limiter *rate.Limiter
}

func (r *TimeoutReader) ReadMultiBuffer() (buf.MultiBuffer, error) {
	mb, err := r.Reader.ReadMultiBuffer()
	if err != nil {
		return nil, err
	}
	if mb.Len() > 0 {
		if err := r.Limiter.WaitN(context.Background(), int(mb.Len())); err != nil {
			buf.ReleaseMulti(mb)
			return nil, err
		}
	}
	return mb, nil
}

func (r *TimeoutReader) ReadMultiBufferTimeout(timeout time.Duration) (buf.MultiBuffer, error) {
	mb, err := r.Reader.ReadMultiBufferTimeout(timeout)
	if err != nil {
		return nil, err
	}
	if mb.Len() > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		if err := r.Limiter.WaitN(ctx, int(mb.Len())); err != nil {
			buf.ReleaseMulti(mb)
			return nil, err
		}
	}
	return mb, nil
}

func (r *TimeoutReader) Interrupt() { common.Interrupt(r.Reader) }

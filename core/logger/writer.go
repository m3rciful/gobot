package logger

import (
	"bufio"
	"errors"
	"io"
	"sync"
)

// asyncWriter provides buffered asynchronous writes to one or more sinks.
type asyncWriter struct {
	queue    chan []byte
	flushReq chan chan error
	done     chan struct{}
	once     sync.Once
	sinks    []*bufio.Writer
	sinkMu   sync.Mutex
	writeErr error
}

func newAsyncWriter(writers []io.Writer, bufSize int) *asyncWriter {
	if bufSize <= 0 {
		bufSize = 64 * 1024
	}
	sinks := make([]*bufio.Writer, 0, len(writers))
	for _, w := range writers {
		if w == nil {
			continue
		}
		sinks = append(sinks, bufio.NewWriterSize(w, bufSize))
	}
	aw := &asyncWriter{
		queue:    make(chan []byte, 256),
		flushReq: make(chan chan error),
		done:     make(chan struct{}),
		sinks:    sinks,
	}
	go aw.loop()
	return aw
}

func (w *asyncWriter) loop() {
	for {
		select {
		case data, ok := <-w.queue:
			if !ok {
				w.flushAll()
				close(w.done)
				return
			}
			if len(data) == 0 {
				continue
			}
			if err := w.writeAll(data); err != nil {
				w.setErr(err)
			}
		case ack := <-w.flushReq:
			ack <- w.flushAll()
		}
	}
}

// Write enqueues the payload for asynchronous fan-out to all sinks.
func (w *asyncWriter) Write(p []byte) error {
	if err := w.getErr(); err != nil {
		return err
	}
	if len(p) == 0 {
		return nil
	}
	data := make([]byte, len(p))
	copy(data, p)
	select {
	case w.queue <- data:
		return nil
	default:
		// queue full; fall back to blocking write to preserve logs
		w.queue <- data
		return nil
	}
}

// Flush waits for the writer to flush all buffered content to sinks.
func (w *asyncWriter) Flush() error {
	if err := w.getErr(); err != nil {
		return err
	}
	ack := make(chan error, 1)
	w.flushReq <- ack
	return <-ack
}

// Close drains the queue and reports the first encountered write error.
func (w *asyncWriter) Close() error {
	w.once.Do(func() {
		close(w.queue)
	})
	<-w.done
	return w.getErr()
}

func (w *asyncWriter) writeAll(p []byte) error {
	w.sinkMu.Lock()
	defer w.sinkMu.Unlock()
	for _, sink := range w.sinks {
		if sink == nil {
			continue
		}
		if _, err := sink.Write(p); err != nil {
			return err
		}
		if err := sink.Flush(); err != nil {
			return err
		}
	}
	return nil
}

func (w *asyncWriter) flushAll() error {
	w.sinkMu.Lock()
	defer w.sinkMu.Unlock()
	var errs []error
	for _, sink := range w.sinks {
		if sink == nil {
			continue
		}
		if err := sink.Flush(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (w *asyncWriter) getErr() error {
	w.sinkMu.Lock()
	defer w.sinkMu.Unlock()
	return w.writeErr
}

func (w *asyncWriter) setErr(err error) {
	if err == nil {
		return
	}
	w.sinkMu.Lock()
	defer w.sinkMu.Unlock()
	if w.writeErr == nil {
		w.writeErr = err
	}
}

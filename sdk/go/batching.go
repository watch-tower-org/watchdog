package wt

import (
	"context"
	"log"
	"sync"
	"time"
)

// retryBackoff is the fixed delay between retry attempts of a failed batch.
const retryBackoff = 500 * time.Millisecond

// batch buffers reported events and flushes them to the backend on an interval
// or once the batch size is reached. Enqueue never blocks: if the buffer is
// full the event is dropped and logged, so error reporting can never stall the
// host application (a core design goal of the SDK).
type batch struct {
	mu        sync.Mutex
	buf       []Event
	closed    bool
	interval  time.Duration
	batchSize int
	maxQueue  int
	sink      Sender
	timeout   time.Duration
	maxRetries   int
	retryBackoff time.Duration
	logger    *log.Logger

	sendMu sync.Mutex
	stop   chan struct{}
	done   chan struct{}
}

func newBatch(sink Sender, interval time.Duration, batchSize, maxQueue int, timeout time.Duration, maxRetries int, retryBackoff time.Duration, logger *log.Logger) *batch {
	b := &batch{
		buf:           make([]Event, 0, batchSize),
		interval:      interval,
		batchSize:     batchSize,
		maxQueue:      maxQueue,
		sink:          sink,
		timeout:       timeout,
		maxRetries:    maxRetries,
		retryBackoff:  retryBackoff,
		logger:        logger,
		stop:          make(chan struct{}),
		done:          make(chan struct{}),
	}
	go b.run()
	return b
}

// run flushes buffered events on every tick until the batch is stopped.
func (b *batch) run() {
	ticker := time.NewTicker(b.interval)
	defer ticker.Stop()
	defer close(b.done)
	for {
		select {
		case <-ticker.C:
			b.flush()
		case <-b.stop:
			return
		}
	}
}

// enqueue buffers an event. It returns false when the batch is closed or the
// queue is full (in which case the event is dropped).
func (b *batch) enqueue(e Event) bool {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return false
	}
	if len(b.buf) >= b.maxQueue {
		b.mu.Unlock()
		b.logger.Printf("watchtower: queue full, dropping event")
		return false
	}
	b.buf = append(b.buf, e)
	full := len(b.buf) >= b.batchSize
	b.mu.Unlock()
	if full {
		go b.flush()
	}
	return true
}

// drain takes ownership of the buffered events.
func (b *batch) drain() []Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.buf) == 0 {
		return nil
	}
	events := b.buf
	b.buf = nil
	return events
}

// flush sends the buffered events in one request. Only one send runs at a
// time. A failed send is retried up to maxRetries additional times with a
// short backoff (in the flusher goroutine, so callers never block); after that
// the batch is dropped and logged.
func (b *batch) flush() {
	events := b.drain()
	if len(events) == 0 {
		return
	}
	b.sendMu.Lock()
	defer b.sendMu.Unlock()

	var lastErr error
	attempts := 0
	for {
		ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
		_, err := b.sink.Send(ctx, events)
		cancel()
		if err == nil {
			return
		}
		lastErr = err
		attempts++
		if attempts > b.maxRetries {
			break
		}
		time.Sleep(b.retryBackoff)
	}
	b.logger.Printf("watchtower: failed to send %d event(s) after %d attempt(s): %v", len(events), attempts, lastErr)
}

// Flush synchronously sends any buffered events.
func (b *batch) Flush() {
	b.flush()
}

// Close flushes remaining events and stops the flusher goroutine. Safe to call
// multiple times; subsequent enqueues are rejected.
func (b *batch) Close() {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return
	}
	b.closed = true
	b.mu.Unlock()
	close(b.stop)
	b.flush()
	<-b.done
}

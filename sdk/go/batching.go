package wt

import (
	"context"
	"log"
	"sync"
	"time"
)

// batch buffers reported events and flushes them to the backend on an interval
// or once the batch size is reached. Enqueue never blocks: if the buffer is
// full the event is dropped and logged, so error reporting can never stall the
// host application (a core design goal of the SDK).
type batch struct {
	mu        sync.Mutex
	buf       []event
	closed    bool
	interval  time.Duration
	batchSize int
	maxQueue  int
	tr        *transport
	logger    *log.Logger

	sendMu sync.Mutex
	stop   chan struct{}
	done   chan struct{}
}

func newBatch(tr *transport, interval time.Duration, batchSize, maxQueue int, logger *log.Logger) *batch {
	b := &batch{
		buf:       make([]event, 0, batchSize),
		interval:  interval,
		batchSize: batchSize,
		maxQueue:  maxQueue,
		tr:        tr,
		logger:    logger,
		stop:      make(chan struct{}),
		done:      make(chan struct{}),
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
func (b *batch) enqueue(e event) bool {
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
func (b *batch) drain() []event {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.buf) == 0 {
		return nil
	}
	events := b.buf
	b.buf = nil
	return events
}

// flush sends the buffered events in one request. Only one HTTP send runs at a
// time; a failed send is logged and dropped (no retry).
func (b *batch) flush() {
	events := b.drain()
	if len(events) == 0 {
		return
	}
	b.sendMu.Lock()
	defer b.sendMu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), b.tr.client.Timeout)
	defer cancel()
	if _, err := b.tr.send(ctx, events); err != nil {
		b.logger.Printf("watchtower: failed to send %d event(s): %v", len(events), err)
	}
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

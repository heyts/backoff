// Package backoff implements a simple backoff scheme to allow a function to be executed periodically until it return a valid result or fails after n attempts.
//
// The package implement both linear and exponential backoff, with a configurable number of retries and delay between retries.
//
// Optionally, a callback function can be executed on successful function invocation.
package backoff

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"
)

// Func is
type Func func() (result interface{}, err error)

// CallbackFunc is
type CallbackFunc func(b Backoff, r interface{})

// Backoff is
type Backoff struct {
	backoffFunc   Func
	callbackFunc  CallbackFunc
	maxRetries    uint
	retryAfter    uint
	exponential   bool
	label         string
	exitOnFailure bool
	log           *log.Logger
	timeScale     time.Duration

	invocations       uint
	failedInvocations uint
}

// Exponential is
func Exponential(f Func, l string) *Backoff {
	label := fmt.Sprintf("%v: ", l)
	log := log.New(os.Stdout, label, log.LstdFlags)

	return &Backoff{
		backoffFunc:   f,
		callbackFunc:  nil,
		maxRetries:    10,
		retryAfter:    500,
		exponential:   true,
		label:         l,
		exitOnFailure: false,
		timeScale:     time.Millisecond,
		log:           log,
	}
}

// Linear is
func Linear(f Func, l string) *Backoff {
	label := fmt.Sprintf("%v: ", l)
	log := log.New(os.Stdout, label, log.LstdFlags)

	return &Backoff{
		backoffFunc:   f,
		callbackFunc:  nil,
		maxRetries:    10,
		retryAfter:    500,
		exponential:   false,
		label:         l,
		exitOnFailure: false,
		timeScale:     time.Millisecond,
		log:           log,
	}
}

// WithRetries is
func (b *Backoff) WithRetries(nbr uint) *Backoff {
	b.maxRetries = nbr
	return b
}

// WithDelay is
func (b *Backoff) WithDelay(ms uint) *Backoff {
	b.retryAfter = ms
	return b
}

// WithTimeScale is
func (b *Backoff) WithTimeScale(t time.Duration) *Backoff {
	b.timeScale = t
	return b
}

// WithExitOnFailure is
func (b *Backoff) WithExitOnFailure(e bool) *Backoff {
	b.exitOnFailure = e
	return b
}

// WithCallback is
func (b *Backoff) WithCallback(c CallbackFunc) *Backoff {
	b.callbackFunc = c
	return b
}

// WithLogger is
func (b *Backoff) WithLogger(w io.Writer) *Backoff {
	b.log = log.New(w, b.label, log.LstdFlags)
	return b
}

// Exec is
func (b *Backoff) Exec() (result interface{}, err error) {
	var prevErr error
	var i uint
	for i = 1; i <= b.maxRetries; i++ {
		var d uint
		b.invocations = i

		if b.exponential {
			d = b.retryAfter * i
		} else {
			d = b.retryAfter
		}

		time.Sleep(time.Duration(d) * b.timeScale)
		result, err = b.backoffFunc()
		if err != nil {
			b.failedInvocations++
			b.log.Printf("(Attempt #%v): %v\n", i, err)
			prevErr = err
			err = nil
			continue
		}
		break
	}

	if result == nil && b.exitOnFailure {
		b.log.Fatalf("giving up after %d tries", b.maxRetries)
	}

	if b.callbackFunc != nil && prevErr == nil {
		b.callbackFunc(*b, result)
	}
	return result, prevErr
}

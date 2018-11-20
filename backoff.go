// Package backoff implements a simple backoff algorithm, executing
// a function repeatedly until it returns a non-error result or
// the maximum allowed number of retries has been reached.
package backoff

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"runtime"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	minRetries = 1
	maxRetries = 100
)

var (
	// ErrInvalidRetriesNumber is an error returned when the number of retries is invalid
	ErrInvalidRetriesNumber = errors.New("invalid number of retries")
)

// UnrecoverableError is a type that wraps an error
// to signal that the error cannot be recovered
// and the backoff function should return immediately
type UnrecoverableError struct{ error }

// NewUnrecoverableError creates a new instance of UnrecoverableError
// which can wrap any error
func NewUnrecoverableError(err error) *UnrecoverableError {
	return &UnrecoverableError{err}
}

type backoffConfig struct {
	backoffFunc  Func
	callbackFunc CallbackFunc
	maxRetries   uint
	retryAfter   uint
	jitterFunc   JitterFunc
	exponential  bool
	label        string
	log          *log.Logger
	timeScale    time.Duration

	invocations       uint
	failedInvocations uint
}

// Func is the function type being wrapped by the backoff function, returning a result and an error.
type Func func() (result interface{}, err error)

// CallbackFunc is the function type to be used as a callback on backoff success
type CallbackFunc func(b *backoffConfig, r interface{})

// ConfigFunc is the function being used to modify the default configuration of the main backoff functions
type ConfigFunc func(b *backoffConfig) error

// JitterFunc is a function that takes an uint and returns an int of the same
// value with added jitter
type JitterFunc func(cap uint) int

// Linear execute the function f repeatedly, until its result is non-nil and no error is returned.
// It keeps the time between each iteration constant.
//
// A result and an error are both returned as soon as the function f returns a non-nil
// result or if the maximum allowed number of retries has been reached.
func Linear(f Func, opts ...ConfigFunc) (interface{}, error) {
	label := getLabel(f)
	cfg := &backoffConfig{
		backoffFunc:  f,
		callbackFunc: nil,
		maxRetries:   10,
		retryAfter:   500,
		jitterFunc:   FullJitter,
		exponential:  false,
		label:        label,
		timeScale:    time.Millisecond,
		log:          log.New(),
	}

	for _, opt := range opts {
		err := opt(cfg)
		if err != nil {
			return nil, err
		}
	}

	return exec(f, cfg)
}

// MustLinear execute the function f repeatedly, until its result is non-nil and no error is returned.
// It keeps the time between each iteration constant.
//
// A result is returned as soon as the function f returns a non-nil result.
// If the function is still failing after the retry attempts, Panic is triggered.
func MustLinear(f Func, opts ...ConfigFunc) interface{} {
	label := getLabel(f)
	cfg := &backoffConfig{
		backoffFunc:  f,
		callbackFunc: nil,
		maxRetries:   10,
		retryAfter:   500,
		exponential:  false,
		label:        label,
		timeScale:    time.Millisecond,
		log:          log.New(),
	}

	for _, opt := range opts {
		err := opt(cfg)
		if err != nil {
			cfg.log.Fatal(err)
		}
	}

	return mustExec(f, cfg)
}

// Exponential execute the function f repeatedly, until its result is non-nil and no error is returned.
// It increases the time between retries after each iteration.
//
// A result and an error are both returned as soon as the function f returns a non-nil
// result or if the maximum allowed number of retries has been reached.
func Exponential(f Func, opts ...ConfigFunc) (interface{}, error) {
	label := getLabel(f)
	cfg := &backoffConfig{
		backoffFunc:  f,
		callbackFunc: nil,
		maxRetries:   10,
		retryAfter:   500,
		exponential:  true,
		label:        label,
		timeScale:    time.Millisecond,
		log:          log.New(),
	}

	for _, opt := range opts {
		err := opt(cfg)
		if err != nil {
			return nil, err
		}
	}

	return exec(f, cfg)
}

// MustExponential execute the function f repeatedly, until its result is non-nil and no error is returned.
// It increases the time between retries after each iteration.
//
// A result and an error are both returned as soon as the function f returns a non-nil
// result or if the maximum allowed number of retries has been reached.
func MustExponential(f Func, opts ...ConfigFunc) interface{} {
	label := getLabel(f)
	cfg := &backoffConfig{
		backoffFunc:  f,
		callbackFunc: nil,
		maxRetries:   10,
		retryAfter:   500,
		exponential:  true,
		label:        label,
		timeScale:    time.Millisecond,
		log:          log.New(),
	}

	for _, opt := range opts {
		err := opt(cfg)
		if err != nil {
			cfg.log.Fatal(err)
		}
	}

	return mustExec(f, cfg)
}

// Retries is a configuration option that sets the number of retries to attempt before giving up.
func Retries(n uint) ConfigFunc {
	return func(b *backoffConfig) error {
		if n > 100 || n < 0 {
			return ErrInvalidRetriesNumber
		}
		b.maxRetries = n
		return nil
	}
}

// Label is a configuration option that sets a custom label for the backoff function.
// The label is by default the name of the package and the name of the function, separated by a colon.
// The label is used mainly as a log prefix, to clarify which function is the subject of logging.
func Label(label string) ConfigFunc {
	return func(b *backoffConfig) error {
		b.label = label
		b.log = log.New()
		return nil
	}
}

// RetryAfter is a configuration option that sets the number of milliseconds (by default)
// to wait before retrying the function execution.
//
// The configuration option `TimeScale` can be used to change the duration unit.
func RetryAfter(n uint) ConfigFunc {
	return func(b *backoffConfig) error {
		b.maxRetries = n
		return nil
	}
}

// TimeScale is a configuration option that sets the timescale for the Backoff function to operate on, as a time.Duration.
// In practice this is mainly used in tests to shorten the time taken by the tests, substituting time.Milliseconds by time.Nanoseconds.
func TimeScale(t time.Duration) ConfigFunc {
	return func(b *backoffConfig) error {
		b.timeScale = t
		return nil
	}
}

// Logger is a configuration option that sets the destination of logging. Practically it expects an io.Writer for destination
func Logger(dest *log.Logger) ConfigFunc {
	return func(b *backoffConfig) error {
		b.log = dest
		return nil
	}
}

// Callback is a configuration option that sets the callback function for the backoff function.
// The callback function is executed when the wrapped function returns a result, completing the backoff
func Callback(f CallbackFunc) ConfigFunc {
	return func(b *backoffConfig) error {
		b.callbackFunc = f
		return nil
	}
}

// Jitter is a configuration option that sets the callback function for the backoff function.
func Jitter(f JitterFunc) ConfigFunc {
	return func(b *backoffConfig) error {
		b.jitterFunc = f
		return nil
	}
}

// Credit: https://play.golang.org/p/Dyj99EjRVm
func getLabel(f Func) string {
	var label string
	v := reflect.ValueOf(f)
	if v.Kind() == reflect.Func {
		if rf := runtime.FuncForPC(v.Pointer()); rf != nil {
			label = rf.Name()
		}
	} else {
		label = v.String()
	}
	s := strings.Split(label, "/")
	label = s[len(s)-1]
	pkg := strings.Split(label, ".")[0]
	return pkg
}

func exec(f Func, b *backoffConfig) (result interface{}, err error) {
	var prevErr error
	var i uint
	for i = 1; i <= b.maxRetries; i++ {
		var d uint
		b.invocations = i

		if b.exponential {
			d = uint(b.jitterFunc(b.retryAfter) * int(i))
		} else {
			d = uint(b.jitterFunc(b.retryAfter))
		}

		time.Sleep(time.Duration(d) * b.timeScale)
		result, err = b.backoffFunc()
		if err != nil {
			switch err.(type) {
			case *UnrecoverableError, UnrecoverableError:
				return int64(0), err

			default:
				b.log.Warnf("%v (Attempt #%v): %v", b.label, i, err)
				b.failedInvocations++
				prevErr = err
				err = nil
				continue
			}
		}
		prevErr = nil
		break
	}
	if b.callbackFunc != nil && prevErr == nil {
		b.callbackFunc(b, result)
	}
	return result, prevErr
}

func mustExec(f Func, b *backoffConfig) (result interface{}) {
	var err error
	var i uint
	for i = 1; i <= b.maxRetries; i++ {
		var d uint
		b.invocations = i

		if b.exponential {
			d = uint(b.jitterFunc(b.retryAfter) * int(i))
		} else {
			d = uint(b.jitterFunc(b.retryAfter))
		}

		time.Sleep(time.Duration(d) * b.timeScale)
		result, err = b.backoffFunc()
		if err != nil {
			switch err.(type) {
			case *UnrecoverableError, UnrecoverableError:
				panic(fmt.Sprintf("giving up after %d tries", b.maxRetries))

			default:
				b.log.Warnf("%v (Attempt #%v): %v", b.label, i, err)
				b.failedInvocations++
				continue
			}
		}
		break
	}

	if err != nil {
		panic(fmt.Sprintf("giving up after %d tries", b.maxRetries))
	}

	if b.callbackFunc != nil {
		b.callbackFunc(b, result)
	}
	return result
}

// NoJitter returns the number passed to it, representing a noop implementation of jitter
func NoJitter(cap uint) int {
	return int(cap)
}

// FullJitter returns a number between 0 and the number passed
func FullJitter(cap uint) int {
	return rand.Intn(int(cap))
}

// EqualJitter return cap/2 add a random jitter in the range 0 to cap/2
func EqualJitter(cap uint) int {
	half := int(math.Floor(float64(cap / 2)))
	return half + rand.Intn(half)
}

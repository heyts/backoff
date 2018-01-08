package backoff_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/heyts/backoff"
)

func Example() {
	testFunc := func() (result interface{}, err error) {
		return true, nil
	}

	res, err := backoff.Exponential(testFunc)
	fmt.Printf("%v, %v", res, err)
	// Output:true, <nil>
}

func Example_simple() {
	testFunc := func() (result interface{}, err error) {
		return true, nil
	}

	res, err := backoff.Exponential(testFunc)
	fmt.Printf("%v, %v", res, err)
	// Output:true, <nil>
}

var errSample = errors.New("There was a problem")

func failingFunc() (result interface{}, err error) {
	return nil, errSample
}

func successFunc() (result interface{}, err error) {
	return "Success", nil
}

func successAfterFunc(n int) (f func() (interface{}, error)) {
	var i int
	return func() (result interface{}, err error) {
		if i < n {
			i++
			return nil, errSample
		}
		return "Success", nil
	}
}

func TestSuccessLinear(t *testing.T) {
	result, err := backoff.Linear(
		successFunc,
		backoff.Retries(5),
	)

	if result != "Success" {
		t.Errorf("Expected result to be \"Success\" but found %v", result)
	}

	if err != nil {
		t.Errorf("Expected error to be nil but found %v", err)
	}
}

func TestFailingLinear(t *testing.T) {
	result, err := backoff.Linear(
		failingFunc,
		backoff.Retries(3),
		backoff.TimeScale(time.Nanosecond),
		backoff.Log(ioutil.Discard),
	)

	if result != nil {
		t.Errorf("Expected result to be nil but found %v", result)
	}

	if err == nil {
		t.Errorf("Expected error to be nil but found %v", err)
	}
}

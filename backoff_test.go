package backoff

import (
	"errors"
	"fmt"
	"io/ioutil"
	"testing"
	"time"
)

func Example() {
	test := func() (result interface{}, err error) {
		return true, nil
	}

	res, err := Exponential(test, "example").WithRetries(6).WithDelay(800).Exec()
	fmt.Printf("%v, %v", res, err)
	// Output:true, <nil>
}

func Example_simple() {
	test := func() (result interface{}, err error) {
		return true, nil
	}

	res, err := Exponential(test, "example").WithRetries(6).WithDelay(800).Exec()
	fmt.Printf("%v, %v", res, err)
	// Output:true, <nil>
}

var errSample = errors.New("There was a problem")

func failing() (result interface{}, err error) {
	return nil, errSample
}

func success() (result interface{}, err error) {
	return "Success", nil
}

func successAfter(n int) (f func() (interface{}, error)) {
	var i int
	return func() (result interface{}, err error) {
		if i < n {
			i++
			return nil, errSample
		} else {
			return "Success", nil
		}
	}
}

func TestFailingFunctionInvocation(t *testing.T) {
	res, err := Exponential(failing, "FailingFunc").WithTimeScale(time.Nanosecond).WithLogger(ioutil.Discard).WithRetries(2).Exec()

	if res != nil {
		t.Errorf("Expected result to be nil, found %v instead", res)
	}

	if err == nil {
		t.Errorf("Expected error to be %q, found %v instead", errSample, err)
	}
}

package backoff

import (
	"errors"
	"fmt"
	"io/ioutil"
	"testing"
	"time"
)

func Example() {
	testFunc := func() (result interface{}, err error) {
		return true, nil
	}

	res, err := Exponential(testFunc)
	fmt.Printf("%v, %v", res, err)
	// Output:true, <nil>
}

func Example_simple() {
	testFunc := func() (result interface{}, err error) {
		return true, nil
	}

	res, err := Exponential(testFunc)
	fmt.Printf("%v, %v", res, err)
	// Output:true, <nil>
}

var errSample = errors.New("There was a problem")

func FailingConfig(m string) ConfigFunc {
	return func(b *backoffConfig) error {
		return fmt.Errorf("%v", m)
	}
}

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

func TestSuccessMustLinear(t *testing.T) {
	result := MustLinear(
		successFunc,
		Retries(5),
		TimeScale(time.Nanosecond),
		Log(ioutil.Discard),
	)

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Expected MustLinear not to panic but it did")
		}
	}()

	if result != "Success" {
		t.Errorf("Expected result to be \"Success\" but found %v", result)
	}
}

func TestFailingMustLinear(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected MustLinear to panic but it didn't")
		}
	}()

	result := MustLinear(
		failingFunc,
		Retries(3),
		TimeScale(time.Nanosecond),
		Log(ioutil.Discard),
	)

	if result != nil {
		t.Errorf("Expected result to be nil but found %v", result)
	}
}

func TestFailingConfigFunc(t *testing.T) {
	result, err := Linear(
		successFunc,
		FailingConfig("Failing"),
		Log(ioutil.Discard),
	)

	if result != nil {
		t.Errorf("Expected result to nil but found %v", result)
	}

	if err.Error() != "Failing" {
		t.Errorf("Expected an error but found %v", err)
	}
}

func TestSuccessLinear(t *testing.T) {
	result, err := Linear(
		successFunc,
		Retries(3),
		TimeScale(time.Nanosecond),
		Log(ioutil.Discard),
	)

	if result != "Success" {
		t.Errorf("Expected result to be \"Success\" but found %v", result)
	}

	if err != nil {
		t.Errorf("Expected error to be nil but found %v", err)
	}
}

func TestFailingLinear(t *testing.T) {
	result, err := Linear(
		failingFunc,
		Retries(3),
		TimeScale(time.Nanosecond),
		Log(ioutil.Discard),
	)

	if result != nil {
		t.Errorf("Expected result to be nil but found %v", result)
	}

	if err == nil {
		t.Errorf("Expected error to be nil but found %v", err)
	}
}

func TestSuccessExponential(t *testing.T) {
	result, err := Exponential(
		successFunc,
		Retries(3),
		TimeScale(time.Nanosecond),
		Log(ioutil.Discard),
	)

	if result != "Success" {
		t.Errorf("Expected result to be \"Success\" but found %v", result)
	}

	if err != nil {
		t.Errorf("Expected error to be nil but found %v", err)
	}
}

func TestFailingExponential(t *testing.T) {
	result, err := Exponential(
		failingFunc,
		Retries(3),
		TimeScale(time.Nanosecond),
		Log(ioutil.Discard),
	)

	if result != nil {
		t.Errorf("Expected result to be nil but found %v", result)
	}

	if err == nil {
		t.Errorf("Expected error to be nil but found %v", err)
	}
}

func TestSuccessMustExponential(t *testing.T) {
	result := MustExponential(
		successFunc,
		Retries(3),
		TimeScale(time.Nanosecond),
		Log(ioutil.Discard),
	)

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Expected MustLinear not to panic but it did")
		}
	}()

	if result != "Success" {
		t.Errorf("Expected result to be \"Success\" but found %v", result)
	}
}

func TestFailingMustExponential(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected MustLinear to panic but it didn't")
		}
	}()

	result := MustExponential(
		failingFunc,
		Retries(3),
		TimeScale(time.Nanosecond),
		Log(ioutil.Discard),
	)

	if result != nil {
		t.Errorf("Expected result to be nil but found %v", result)
	}
}

func TestInvalidRetries(t *testing.T) {
	_, err := Linear(
		failingFunc,
		Retries(329),
		TimeScale(time.Nanosecond),
	)
	if err != ErrInvalidRetriesNumber {
		t.Errorf("Invalid retries configuration option should return an error")
	}
}

func TestCallback(t *testing.T) {

	cb := func(b *backoffConfig, r interface{}) {
		fmt.Printf("Callback")
	}

	CallbackInterceptor := func() ConfigFunc {
		return func(b *backoffConfig) error {
			if b.callbackFunc == nil {
				t.Errorf("Expected Callback to be set")
			}
			return nil
		}
	}

	result, err := Linear(
		successFunc,
		Retries(3),
		TimeScale(time.Nanosecond),
		Log(ioutil.Discard),
		Callback(cb),
		CallbackInterceptor(),
	)

	if result != "Success" {
		t.Errorf("Expected result to be \"Success\" but found %v", result)
	}

	if err != nil {
		t.Errorf("Expected error to be nil but found %v", err)
	}
}

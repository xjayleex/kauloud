package kauloud

import (
	"fmt"
	"strings"
)

var  (
	ErrorTypeAssertionForPod = Error{ msg : "error on type assertion for pod" }
	ErrorTypeAssertionForNode = Error{ msg : "error on type assertion for node" }
	ErrorPodPendingWithNoNode = Error{ msg : "pod is pending with no node name" }
)

type Error struct {
	err error
	msg string
}

func (err Error) Error() string {
	if err.err != nil {
		return fmt.Sprintf("%s: %v", err.msg, err.err)
	}
	return err.msg
}

func (err Error) wrap(inner error) error {
	return Error{msg: err.msg, err: inner}
}

func (err Error) Unwrap() error {
	return err.err
}

func (err Error) Is(target error) bool {
	ts := target.Error()
	return ts == err.msg || strings.HasPrefix(ts, err.msg + ": ")
}
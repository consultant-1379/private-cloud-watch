package log

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/Sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestLogging(t *testing.T) {
	// Testing that log creation works and adds callerFunc
	logEntry := WithField("foo", "bar")
	val, ok := logEntry.Data["foo"]
	assert.Equal(t, true, ok)
	assert.Equal(t, "bar", val)
	caller, ok := logEntry.Data[callerFunc]
	assert.Equal(t, true, ok)
	assert.Equal(t, "g/e/m/p/u/log.TestLogging", caller)

	// Testing that logging at full verbosity works and adds the fields
	// 1) callerFunc
	// 2) callerFile
	// 3) callerLine
	var b bytes.Buffer
	SetOutput(&b)
	SetFormatter(new(logrus.JSONFormatter))
	err := SetVerbosity(2)
	assert.Nil(t, err)
	errMsg := "this is an error"
	Error(errMsg) // this is line 33, update assertion below if diff
	var fields logrus.Fields
	err = json.Unmarshal(b.Bytes(), &fields)
	assert.Nil(t, err)
	assert.Equal(t, 6, len(fields))
	assert.Equal(t, errMsg, fields["msg"])
	assert.Equal(t, "github.com/erixzone/myriad/pkg/util/log.TestLogging",
		fields["Func"])
	assert.Equal(t, "33", fmt.Sprint(fields["Line"]))

	// Testing that logging at 0 verbosity still logs the message, but
	// does not add any other fields
	b.Reset()
	err = SetVerbosity(0)
	assert.Nil(t, err)
	Error(errMsg)
	fields = logrus.Fields{}
	err = json.Unmarshal(b.Bytes(), &fields)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(fields))
	assert.Equal(t, errMsg, fields["msg"])
}

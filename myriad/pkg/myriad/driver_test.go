package myriad

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"

	"github.com/erixzone/myriad/pkg/myriadca"
	"github.com/erixzone/myriad/pkg/myriadfile"
)

type testFakeDriver struct{}
type testFakeFactory struct{}

func (f testFakeFactory) New() (Driver, error) {
	return &testFakeDriver{}, nil
}

func (t *testFakeDriver) Run(
	ctx context.Context,
	jobs []myriadfile.Job,
	ca *myriadca.CertificateAuthority) (err error) {
	return nil
}

func TestRegistration(t *testing.T) {
	var d Driver
	var err error

	d, err = GetDriver("foo")
	assert.Nil(t, d)
	assert.NotNil(t, err)

	err = RegisterDriver("foo", testFakeFactory{})
	assert.Nil(t, err)

	d, err = GetDriver("foo")
	assert.Nil(t, err)
	assert.NotNil(t, d)

	err = d.Run(context.Background(), []myriadfile.Job{}, nil)
	assert.Nil(t, err)

	err = RegisterDriver("foo", testFakeFactory{})
	assert.NotNil(t, err)
}

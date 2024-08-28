package myriad

// Copyright 2016 Ericsson AB All Rights Reserved.
// Contributors:
//     Andrew Hume
//     Scott Devoid

import (
	"fmt"

	"golang.org/x/net/context"

	"github.com/erixzone/myriad/pkg/myriadca"
	"github.com/erixzone/myriad/pkg/myriadfile"
)

var registered map[string]DriverFactory

func init() {
	registered = make(map[string]DriverFactory)
}

// Driver defines the interface that myriad drivers use to run myriad.
// Run is the function that the driver implements to actually run a set
// of myriad jobs as defined in a myriadfile.
type Driver interface {
	Run(ctx context.Context, jobs []myriadfile.Job, ca *myriadca.CertificateAuthority) (err error)
}

// DriverFactory constructs an instance of a driver.
type DriverFactory interface {
	New() (Driver, error)
}

// RegisterDriver is called my myriad drivers in an init() function to
// register themselves with myriad. The driver package should also be
// 'underscore' imported in cmd/root.go to trigger this side-effect.
// If a driver has already been registered with this name this function
// returns an error.
func RegisterDriver(name string, f DriverFactory) (err error) {
	if _, ok := registered[name]; ok {
		return fmt.Errorf("Driver already registered with name '%s'", name)
	}
	registered[name] = f
	return
}

// GetDriver returns the driver registered by 'name' or returns an
// error if no such driver exists.
func GetDriver(name string) (d Driver, err error) {
	fact, ok := registered[name]
	if !ok {
		return nil, fmt.Errorf("Unknown driver '%s'", name)
	}
	return fact.New()
}

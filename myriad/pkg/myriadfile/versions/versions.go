// Copyright 2016 Ericsson AB All Rights Reserved.

// Package versions defines a way to register different myriadfile
// formats and upgrade from older formats to the most recent format.
package versions

import (
	"fmt"
	"strings"
	"sync"
)

var registered map[string]FileParser
var upgradesTo map[string]string
var guard *sync.Mutex

func init() {
	registered = make(map[string]FileParser)
	upgradesTo = make(map[string]string)
	guard = &sync.Mutex{}
}

// FileParser defines methods that allow for the parsing of different versions
// of the myriadfile and upgrading to the latest format. Parse takes a byte-array
// which is the text of the myriadfile and returns a pointer to that parser's representation
// of the myriadfile or an error. Upgrade takes that representation and returns a different
// representation (an upgraded format).
type FileParser interface {
	Parse([]byte) (interface{}, error)
	Upgrade(interface{}) (interface{}, error)
}

// RegisterFormat registers a FileParser for the particular version. The upgradesTo
// indicates what version the passed FileParser upgrades to. If there is no upgrade
// this should be an empty string.
func RegisterFormat(f FileParser, version, upgrades string) error {
	guard.Lock()
	defer guard.Unlock()
	if _, ok := registered[version]; ok {
		msg := "Myriadfile format already registered for version '%s'"
		return fmt.Errorf(msg, version)

	}
	registered[version] = f
	if upgrades != "" {
		upgradesTo[version] = upgrades
	}
	return nil
}

// UpgradeFn is a function which transforms the current representation into
// an updated representation.
type UpgradeFn func(interface{}) (interface{}, error)

func wrapUpgrade(f UpgradeFn, p FileParser) UpgradeFn {
	return func(i interface{}) (interface{}, error) {
		i, err := f(i)
		if err != nil {
			return nil, err
		}
		return p.Upgrade(i)
	}
}

// Upgrade attempts to generate an `UpgradeFn` that will transform a representation
// of the myriadfile in fromVersion to the representation in toVersion. It returns
// an error if this cannot be satisfied or if a cycle is detected.
func Upgrade(fromVersion, toVersion string) (f UpgradeFn, err error) {
	guard.Lock()
	defer guard.Unlock()
	seen := make(map[string]bool)
	var seenStack []string
	for {
		if _, ok := seen[fromVersion]; ok {
			seenStack = append(seenStack, fromVersion)
			stackStr := strings.Join(seenStack, ",")
			msg := "Loop while resolving versions. Upgrade path: %s"
			return nil, fmt.Errorf(msg, stackStr)
		}
		var ok bool
		if fromVersion == toVersion {
			break
		}
		curr, ok := registered[fromVersion]
		if !ok {
			if len(seenStack) == 0 {
				return nil, fmt.Errorf("Unknown version: '%s'", fromVersion)
			}
			msg := "Unable to upgrade myriadfile (version '%s') to version '%s'\n%s"
			return nil, fmt.Errorf(msg, seenStack[0], toVersion, strings.Join(seenStack, ","))
		}
		if f == nil {
			f = func(i interface{}) (interface{}, error) { return curr.Upgrade(i) }
		} else {
			f = wrapUpgrade(f, curr)
		}
		seen[fromVersion] = true
		seenStack = append(seenStack, fromVersion)
		fromVersion, ok = upgradesTo[fromVersion]
	}
	return
}

// GetFileParser returns the fileparser for the specified version or an error.
func GetFileParser(v string) (FileParser, error) {
	guard.Lock()
	defer guard.Unlock()
	p, ok := registered[v]
	if !ok {
		return nil, fmt.Errorf("Unknown version: %s", v)
	}
	return p, nil
}

// Versions returns a list of versions that are registered.
func Versions() (v []string) {
	guard.Lock()
	defer guard.Unlock()
	for name := range registered {
		v = append(v, name)
	}
	return v
}

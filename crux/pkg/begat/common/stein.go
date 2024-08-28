/*
	passing things back from a chore to begat is done through a stein
*/

package common

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

// StringPair is just a simple pair of strings
type StringPair struct {
	Key   string
	Value string
}

// Stein is a collection of data begat needs from a recipe execution
type Stein struct {
	Ofiles []StringPair
	Uvars  []StringPair
}

// NewStein returns a Stein
func NewStein() *Stein {
	var s Stein
	return &s
}

// AddO adds a file and checksum to a stein
func (s *Stein) AddO(file, chk string) {
	s.Ofiles = append(s.Ofiles, StringPair{Key: file, Value: chk})
}

// AddU adds a variable and its value to a stein
func (s *Stein) AddU(name, val string) {
	s.Uvars = append(s.Uvars, StringPair{Key: name, Value: val})
}

// AddToFile adds a stein to an existing file of steins
func (s *Stein) AddToFile(path string) error {
	old, err := FileToStein(path)
	if err != nil {
		return err
	}
	if old != nil {
		s.Ofiles = append(s.Ofiles, old.Ofiles...)
		s.Uvars = append(s.Uvars, old.Uvars...)
	}
	j, err := json.Marshal(*s)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, j, 0644)
}

// FileToStein reads in a file's contents as a stein
func FileToStein(path string) (*Stein, error) {
	b, err := ioutil.ReadFile(path)
	fmt.Printf("readfile(%s) returns %v\n", path, err)
	// TBD handle error of nonexistent file
	var s Stein
	err = json.Unmarshal(b, &s)
	return &s, err
}

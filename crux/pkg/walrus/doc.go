/*
Package walrus is a fork of logrus with Tagged logging and network log-level change features added.
Package logrus is a structured logger for Go, completely API compatible with the standard library logger.


The simplest way to use Logrus/walrus is simply the package-level exported logger:

  package main

  import (
    log "github.com/Sirupsen/logrus"
  )

  func main() {
    log.WithFields(log.Fields{
      "animal": "walrus",
      "number": 1,
      "size":   10,
    }).Info("A walrus appears")
  }

However, package level loggers are not recommended.  Try to create
your packages so that constructors take a logger argument, and use
that logger instead of the local package logger.  Only the main
package will create a logger, and pass it through to the other
packages in the program.

  Output: time="2015-09-07T08:48:33Z" level=info msg="A walrus
appears" animal=walrus number=1 size=10

For a full guide visit https://github.com/Sirupsen/logrus
*/
package walrus

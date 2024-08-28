package main

import "github.com/erixzone/crux/pkg/walrus"

// "os"

var log = walrus.New()

func init() {
	log.Formatter = new(walrus.JSONFormatter)
	log.Formatter = new(walrus.TextFormatter) // default

	// file, err := os.OpenFile("walrus.log", os.O_CREATE|os.O_WRONLY, 0666)
	// if err == nil {
	// 	log.Out = file
	// } else {
	// 	log.Info("Failed to log to file, using default stderr")
	// }

	log.Level = walrus.DebugLevel
}

func main() {
	defer func() {
		err := recover()
		if err != nil {
			log.WithFields(walrus.Fields{
				"omg":    true,
				"err":    err,
				"number": 100,
			}).Fatal("The ice breaks!")
		}
	}()

	log.WithFields(walrus.Fields{
		"animal": "walrus",
		"number": 8,
	}).Debug("Started observing beach")

	log.WithFields(walrus.Fields{
		"animal": "walrus",
		"size":   10,
	}).Info("A group of walrus emerges from the ocean")

	log.WithFields(walrus.Fields{
		"omg":    true,
		"number": 122,
	}).Warn("The group's number increased tremendously!")

	log.WithFields(walrus.Fields{
		"temperature": -4,
	}).Debug("Temperature changes")

	log.WithFields(walrus.Fields{
		"animal": "orca",
		"size":   9009,
	}).Panic("It's over 9000!")
}

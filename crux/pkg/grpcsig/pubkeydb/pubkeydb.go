// (c) Ericsson Inc. 2017 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hogue

// pubkeydb
// Prepares simple BoltDB databases of public keys
// For use by grpcsig server side signature validation

package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/boltdb/bolt"
)

type pubKeyT struct {
	Service    string `json:"service"`
	Name       string `json:"name"`
	KeyID      string `json:"keyid"`
	PubKey     string `json:"pubkey"`
	StateAdded int32  `json:"stateadded"`
}

type endPointT struct {
	NodeID     string `json:"nodeid"`
	NetID      string `json:"netid"`
	Priority   string `json:"priority,omitempty"`
	Rank       int32  `json:"rank,omitempty"`
	StateAdded int32  `json:"stateadded"`
}

var db *bolt.DB

func main() {
	Version := "pubkeydb 2.0"
	pVersion := flag.Bool("v", false, "Bool: Display Version, quit")
	pType := flag.Bool("t", false, "Bool: json io type true = use endpoints, false = use pubkeys")
	pIn := flag.String("i", "pubkeys.json", "File input JSON")
	pOut := flag.String("o", "pubkeys.db", "File output BoltDB")
	pKeys := flag.Bool("k", false, "Bool: Dump key`:` as a prefix to json")
	pAppend := flag.Bool("a", false, "Bool: Append to existing output file")
	pDumpJSON := flag.Bool("j", false, "Bool: Write all BoltDB JSON Values, quit")

	pubkeys := "PubKeys" // Name of bucket...
	endpts := "EndPoints"

	flag.Parse()

	if *pVersion == true {
		fmt.Println(Version)
		os.Exit(0)
	}

	buckname := pubkeys
	if *pType == true {
		buckname = endpts
	}

	dbexists := false
	// Stat the output file, don't use it unless -a=t is specified
	if _, err := os.Stat(*pOut); err == nil {
		dbexists = true
	}

	var derr error
	db, derr = bolt.Open(*pOut, 0600, nil)
	if derr != nil {
		fmt.Fprintf(os.Stderr, "fatal - cannot open target Bolt DB location for writing to file %s [%v]", *pOut, derr)
		os.Exit(1)
	}
	defer db.Close()

	if dbexists && *pDumpJSON == true {
		err := db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte(buckname))
			if b == nil {
				return fmt.Errorf("%s Bucket not found", buckname)
			}
			c := b.Cursor()
			for k, v := c.First(); k != nil; k, v = c.Next() {
				if *pKeys == true {
					fmt.Fprintf(os.Stdout, "%s:%s\n", string(k), string(v))
				} else {
					fmt.Fprintf(os.Stdout, "%s\n", string(v))
				}
			}
			return nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "fatal - json dump failed: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if dbexists && *pAppend == false {
		fmt.Fprintf(os.Stderr, "error - output file already exists: %s\nremove it or specify -a=true to append to it\n", *pOut)
		os.Exit(1)
	}

	derr = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(buckname))
		if err != nil {
			return err
		}
		return nil
	})
	if derr != nil {
		fmt.Fprintf(os.Stderr, "fatal - cannot create Bucket [%s] in BoltDB file: %s\n", buckname, *pOut)
		os.Exit(1)
	}
	fmt.Printf("Loading %s from %s\n", *pOut, *pIn)

	if *pType == false { // pubkeys
		pubkey := pubKeyT{}
		f, err := os.Open(*pIn)
		if err != nil {
			fmt.Fprintf(os.Stderr, "fatal - cannot open input JSON file for reading %s [%v]", *pIn, err)
			os.Exit(1)
		}
		defer f.Close()
		reader := bufio.NewReader(f)
		l := 1
		line, err := reader.ReadString('\n')
		for err == nil {
			if line != "" {
				err := json.Unmarshal([]byte(line), &pubkey)
				if err != nil {
					fmt.Fprintf(os.Stderr, "error - 1 public key json scanning file %s line %d [%v]\n", *pIn, l, err)
				}
				err = db.Update(func(tx *bolt.Tx) error {
					bucket := tx.Bucket([]byte(buckname))
					if bucket == nil {
						return fmt.Errorf("error - bucket %s not found", buckname)
					}
					return bucket.Put([]byte(fmt.Sprintf("%s", pubkey.KeyID)), []byte(line))
				})
				if err != nil {
					fmt.Fprintf(os.Stderr, "%v\n", err)
					os.Exit(1)
				}
			}
			line, err = reader.ReadString('\n')
			l = l + 1
		}
		if err != io.EOF {
			fmt.Fprintf(os.Stderr, "error - 2 public key json scanning file %s line %d [%v]\n", *pIn, l, err)
			os.Exit(1)
		}
	} else { // endpoints
		endpt := endPointT{}
		f, err := os.Open(*pIn)
		if err != nil {
			fmt.Fprintf(os.Stderr, "fatal - annot open input JSON file for reading %s [%v]", *pIn, err)
			os.Exit(1)
		}
		defer f.Close()
		reader := bufio.NewReader(f)
		l := 1
		line, err := reader.ReadString('\n')
		for err == nil {
			if line != "" {
				err := json.Unmarshal([]byte(line), &endpt)
				if err != nil {
					fmt.Fprintf(os.Stderr, "error - 1 endpoint json scanning file %s line %d [%v]\n", *pIn, l, err)
				}
				err = db.Update(func(tx *bolt.Tx) error {
					bucket := tx.Bucket([]byte(buckname))
					if bucket == nil {
						return fmt.Errorf("error - bucket %s not found", buckname)
					}
					return bucket.Put([]byte(fmt.Sprintf("%s%s", endpt.NodeID, endpt.NetID)), []byte(line))
				})
				if err != nil {
					fmt.Fprintf(os.Stderr, "%v\n", err)
					os.Exit(1)
				}
			}
			line, err = reader.ReadString('\n')
			l = l + 1
		}
		if err != io.EOF {
			fmt.Fprintf(os.Stderr, "error - 2 endpoint json scanning file %s line %d [%v]\n", *pIn, l, err)
			os.Exit(1)
		}
	}
	fmt.Println("Finished")
}

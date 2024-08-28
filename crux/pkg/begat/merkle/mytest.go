package merkle

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"
)

// MyTestScenario1 tests a simple tree.
func MyTestScenario1(t *testing.T) {
	RmTree("mx")
	genFile("mx", "goo5", 3, 23)
	genFile("mx", "goo2", 13, 123)
	genFile("mx/sub", "goo3", 1, 68)
	genFile("mx/sub", "goo4", 19, 79)
	genFile("mx", "goo1", 5, 130)
	acts := []string{
		"MerkleChange mx",
		"MerkleChange mx/sub",
		"MerkleNew mx/goo1",
		"MerkleNew mx/goo2",
		"MerkleNew mx/goo5",
		"MerkleNew mx/sub",
		"MerkleNew mx/sub/goo3",
		"MerkleNew mx/sub/goo4",
	}
	MyTestScenario(t, "mx", acts, "7ccae7797a153c2adddc29abf75724fdade503b7b103c1de7d2b1fc9ee66b65f8176d8b5f81086ee5cb4e4f35203d55f6fd89030dd562ddc17ee8cf3686adc33")
}

// MyTestScenario2 tests a new file.
func MyTestScenario2(t *testing.T) {
	genFile("mx/sub", "goo4", 19, 179)
	acts := []string{
		"MerkleChange mx",
		"MerkleChange mx/sub",
		"MerkleChange mx/sub",
		"MerkleChange mx/sub/goo4",
	}
	MyTestScenario(t, "mx", acts, "cdba87119eb2edf2e922eed0f1b44d99766d89ec4a4b76612e1a1b7fe22b0d77d101f631a6f7f8bcda0ff03a05676ee03c7314a165d408709a49aa7add276c07")
}

// MyTestScenario3 tests replacing a file with itself.
func MyTestScenario3(t *testing.T) {
	genFile("mx/sub", "goo4", 19, 179)
	acts := []string{}
	MyTestScenario(t, "mx", acts, "cdba87119eb2edf2e922eed0f1b44d99766d89ec4a4b76612e1a1b7fe22b0d77d101f631a6f7f8bcda0ff03a05676ee03c7314a165d408709a49aa7add276c07")
}

// MyTestScenario4 tests nothing.
func MyTestScenario4(t *testing.T) {
	acts := []string{}
	MyTestScenario(t, "mx", acts, "cdba87119eb2edf2e922eed0f1b44d99766d89ec4a4b76612e1a1b7fe22b0d77d101f631a6f7f8bcda0ff03a05676ee03c7314a165d408709a49aa7add276c07")
}

// MyTestScenario5 tests changing a file.
func MyTestScenario5(t *testing.T) {
	genFile("mx/sub", "goo4", 19, 29)
	acts := []string{
		"MerkleChange mx",
		"MerkleChange mx/sub",
		"MerkleChange mx/sub",
		"MerkleChange mx/sub/goo4",
	}
	MyTestScenario(t, "mx", acts, "203bc6e48214db6e02c65ccabbc823ca65b750ead1642965b28ba9129fe1b54731979f4f892e280fd61cf05a5f665bb219da8345c821dcec102d85dacfccd981")
}

// MyTestScenario6 tests removing a file.
func MyTestScenario6(t *testing.T) {
	os.Remove("mx/sub/goo3")
	acts := []string{
		"MerkleChange mx",
		"MerkleChange mx/sub",
		"MerkleChange mx/sub",
		"MerkleDelete mx/sub/goo3",
	}
	MyTestScenario(t, "mx", acts, "3c531b124cc2ab78e057ed8d4881f08a428f6a639c662f76e9d39e0debd4ded6dcb84469556481fc83c1ef08a77e754c10645ad3207db8ab29635d747a372ca6")
}

// MyTestScenario tests an above example.
func MyTestScenario(t *testing.T, dir string, expActs []string, expHash string) {
	time.Sleep(2 * time.Second)
	ch := make(chan EventMerkle, 10)
	acts := make([]string, 0)
	go func() {
		o, _ := UpdateHashTree(dir, ch, true, false, nil)
		ch <- EventMerkle{Op: MerkleDone, Hash: o.Hash.String(), Ack: nil}
	}()
	for {
		m := <-ch
		if m.Op == MerkleDone {
			break
		}
		s := m.Op.String() + " " + m.File
		acts = append(acts, s)
	}
	m := <-ch
	hash := m.Hash
	sort.Strings(acts)
	same := len(acts) == len(expActs)
	n := len(acts)
	if len(expActs) < n {
		n = len(expActs)
	}
	for i := 0; i < n; i++ {
		if acts[i] != expActs[i] {
			same = false
			break
		}
	}
	if !same {
		fmt.Printf("incorrect acts!! got\n")
		for i, s := range acts {
			fmt.Printf("%d: %s\n", i, s)
		}
		fmt.Printf("but expected\n")
		for i, s := range expActs {
			fmt.Printf("%d: %s\n", i, s)
		}
		t.Fail()
	}
	if hash != expHash {
		fmt.Printf(">>%s<<\n", hash)
		fmt.Printf("inconsistent root hash:\ngot %s\nexp %s\n", hash, expHash)
		t.Fail()
	}
}

// shut up lint!
const (
	KB = 1024
)

func genFile(dir string, entry string, siz int, seed int) {
	os.MkdirAll(dir, 0777)
	path := filepath.Join(dir, entry)
	fmt.Printf("creating %s\n", path)
	file, err := os.Create(path)
	if err != nil {
		// should do something here
		panic(err)
	}
	defer file.Close()
	contents := make([]byte, KB)
	for i := range contents {
		contents[i] = byte(seed)
		seed = (seed + 1) % 256
	}
	wr := bufio.NewWriter(file)
	for siz > 0 {
		wr.Write(contents)
		siz--
	}
	wr.Flush()
}

// (c) Erisson Inc. 2016 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hogue

package auto

import (
	"encoding/json"
	"os"
)

// WorkerFromFile - loads a json worker struct, does not reattach function pointers
func WorkerFromFile(filename string) (WorkerT, error) {
	var w WorkerT
	f, err := os.Open(filename)
	if err != nil {
		return w, err
	}
	defer f.Close()
	parseJSON := json.NewDecoder(f)
	err = parseJSON.Decode(&w)
	if err != nil {
		return w, err
	}
	return w, nil
}

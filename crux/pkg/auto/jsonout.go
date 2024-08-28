// (c) Erisson Inc. 2016 All Rights Reserved.
// Contributors:
//      Christopher W. V. Hogue

package auto

import (
	"encoding/json"
	"fmt"
	"os"
)

// HubToJSON - marshals hub to json
func (h *HubT) HubToJSON() ([]byte, error) {
	hubJSON, err := json.Marshal(h)
	return hubJSON, err
}

// HubToFile - saves hub to a json file
func (h *HubT) HubToFile(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Sync()
	defer f.Close()

	hub, err := h.HubToJSON()
	if err != nil {
		return fmt.Errorf("HUB JSON Marshalling error [%v]", err)
	}
	_, err = f.Write(hub)
	if err != nil {
		return err
	}
	_, err = f.WriteString("\n")
	if err != nil {
		return err
	}
	return nil
}

// WorkerToJSON - marshals worker to json
func (w *WorkerT) WorkerToJSON() ([]byte, error) {
	workerJSON, err := json.Marshal(w)
	return workerJSON, err
}

// WorkerToFile - saves worker to a json file (no function pointers, interfaces)
func (w *WorkerT) WorkerToFile(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Sync()
	defer f.Close()

	worker, err := w.WorkerToJSON()
	if err != nil {
		return fmt.Errorf("Worker JSON Marshalling error [%v]", err)
	}
	_, err = f.Write(worker)
	if err != nil {
		return err
	}
	_, err = f.WriteString("\n")
	if err != nil {
		return err
	}
	return nil
}

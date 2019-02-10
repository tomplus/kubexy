package main

import (
	"encoding/json"
	"net/http"
	"sync"
)

// PodDetails describes a pod in unified format
type PodDetails struct {
	Name      string
	Namespace string
	Node      string
	OwnerKind string
	Status    int
	Size      int
}

// Pod's status constants
const (
	PodStatusInit        = iota // Just created, not scheduled yet
	PodStatusStarting    = iota // Scheduled, containers are starting
	PodStatusRunning     = iota // All containers are up & running
	PodStatusCrash       = iota // One of more containers are down
	PodStatusTerminating = iota // Terminating/deleting
	PodStatusCompleted   = iota // Deleted/completed/evicted
	PodStatusError       = iota // In error state
)

// PodStatusSequenceStart defines list of status for starting pod
var PodStatusSequenceStart = []int{PodStatusInit, PodStatusStarting, PodStatusRunning}

// PodStatusSequenceStop defines list of status for deleting pod
var PodStatusSequenceStop = []int{PodStatusTerminating, PodStatusCompleted}

// PodStatusSequenceCrash define list of status for crashed pod
var PodStatusSequenceCrash = []int{PodStatusCrash, PodStatusStarting, PodStatusRunning}

// ViewPod provides a list of pods with details
type ViewPod struct {
	Pods  map[string]*PodDetails
	mutex sync.Mutex
}

// ServeHTTP serves ViewPod data
func (view *ViewPod) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	view.mutex.Lock()
	defer view.mutex.Unlock()
	if err := json.NewEncoder(w).Encode(view.Pods); err != nil {
		panic(err)
	}

}

// GetFullName returns namespace and pod name
func (pod *PodDetails) GetFullName() string {
	return pod.Namespace + "/" + pod.Name
}

// Equal returns true if pods are the same
func (pod *PodDetails) Equal(podCmp *PodDetails) bool {
	return pod.Name == podCmp.Name &&
		pod.Namespace == podCmp.Namespace &&
		pod.Node == podCmp.Node &&
		pod.OwnerKind == podCmp.OwnerKind &&
		pod.Status == podCmp.Status &&
		pod.Size == podCmp.Size
}

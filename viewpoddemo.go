package main

import (
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"
)

// ViewPodDemo extends ViewPod and store additional parameters related to demo
type ViewPodDemo struct {
	ViewPod
	speed int
	nodes int
	rng   *RandomNameGenerator
}

// NewViewPodDemo inits ViewPodDemo struct with randomized list of pods
func NewViewPodDemo(app KubexyApp) *ViewPodDemo {
	log.Printf("Initialize ViewPod - demo")

	view := ViewPodDemo{
		speed: app.Args.DemoSpeed,
		nodes: app.Args.DemoNodes,
		rng:   NewRandomNameGenerator()}

	view.Pods = make(map[string]*PodDetails)

	// generate pods
	npod := 0
	for npod < app.Args.DemoPods {

		maxRep := app.Args.DemoNodes * 3 / 4
		if app.Args.DemoPods-npod < maxRep {
			maxRep = app.Args.DemoPods - npod
		}

		kinds := []string{"StatefulSet", "ReplicaSet", "DaemonSet"}
		ownerKind := kinds[rand.Intn(len(kinds))]
		namespace := view.rng.GetNamespace(fmt.Sprintf("%v", rand.Intn(app.Args.DemoNamespaces)))
		replicas := 1 + rand.Intn(maxRep)
		size := rand.Intn(90)/replicas + rand.Intn(10)

		prefix := fmt.Sprintf("%v-%x", view.rng.GetPodName(fmt.Sprintf("%v", npod)), rand.Intn(1<<16))

		for nrep := 0; nrep < replicas; nrep++ {
			pod := view.CreatePod(prefix, nrep, namespace, ownerKind, size)

			view.Pods[pod.GetFullName()] = pod
			log.Printf("Generate pod: %v", pod)
			npod++
		}
	}

	// start simulation in the background
	go view.PlayDemo()

	return &view
}

// CreatePod creates PodDetails with randomized node name
func (view *ViewPodDemo) CreatePod(prefix string, seq int, namespace string, owner string, size int) *PodDetails {

	name := fmt.Sprintf("%v-%x", prefix, seq)
	node := view.rng.GetNode(fmt.Sprintf("%v", rand.Intn(view.nodes)))

	return &PodDetails{
		Name:      name,
		Namespace: namespace,
		OwnerKind: owner,
		Node:      node,
		Status:    PodStatusRunning,
		Size:      size}

}

// RecreatePod creates new PodDetails with parameters copied from oldPod
func (view *ViewPodDemo) RecreatePod(prefix string, seq int, oldPod *PodDetails) *PodDetails {

	name := fmt.Sprintf("%v-%x", prefix, seq)
	node := view.rng.GetNode(fmt.Sprintf("%v", rand.Intn(view.nodes)))

	return &PodDetails{
		Name:      name,
		Namespace: oldPod.Namespace,
		OwnerKind: oldPod.OwnerKind,
		Node:      node,
		Status:    PodStatusInit,
		Size:      oldPod.Size}

}

// PlayDemo plays simulation
func (view *ViewPodDemo) PlayDemo() {

	for {
		time.Sleep(time.Duration(view.speed*10) * time.Millisecond)

		switch rndAction := rand.Intn(3); rndAction {
		case 0:
			log.Printf("PlayDemo: crash random container")
			view.DemoCrashContainer()
		case 1:
			log.Printf("PlayDemo: rolling update")
			view.DemoRollingUpdate()
		case 2:
			log.Printf("PlayDemo: recreate")
			view.DemoRecreate()
		default:
			log.Printf("PlayDemo: no action")
		}
	}
}

func (view *ViewPodDemo) randomPod() *PodDetails {
	rndPod := rand.Intn(len(view.Pods))
	i := 0
	for _, pod := range view.Pods {
		if i == rndPod {
			return pod
		}
		i++
	}
	return nil
}

func (view *ViewPodDemo) updateStatus(pod *PodDetails, status []int) {
	for _, newStatus := range status {
		view.mutex.Lock()
		pod.Status = newStatus
		if newStatus == PodStatusCompleted {
			delete(view.Pods, pod.GetFullName())
		} else {
			view.Pods[pod.GetFullName()] = pod
		}
		log.Printf("pod with new status: %v", pod)
		view.mutex.Unlock()
		time.Sleep(time.Duration(view.speed*10) * time.Millisecond)
	}
}

// DemoCrashContainer choses a random pod and simulate crash and start again
func (view *ViewPodDemo) DemoCrashContainer() {

	pod := view.randomPod()
	log.Printf("Random pod: %v", pod)
	view.updateStatus(pod, PodStatusSequenceCrash)
}

// DemoRollingUpdate choses random set of pods and perform rolling update to the new version
func (view *ViewPodDemo) DemoRollingUpdate() {

	pod := view.randomPod()
	log.Printf("Random pod: %v", pod)

	npart := strings.Split(pod.Name, "-")
	newPrefix := fmt.Sprintf("%v-%x", npart[0], rand.Intn(1<<16))

	seq := 0
	for {
		nn := fmt.Sprintf("%v-%v-%x", npart[0], npart[1], seq)
		oldPod, ok := view.Pods[nn]
		if !ok {
			break
		}
		// create new pod
		newPod := view.RecreatePod(newPrefix, seq, oldPod)
		view.updateStatus(newPod, PodStatusSequenceStart)

		// delete old
		view.updateStatus(oldPod, PodStatusSequenceStop)

		seq++
	}

}

// DemoRecreate choses random set of pods and perform recreating pods with new version
func (view *ViewPodDemo) DemoRecreate() {

	pod := view.randomPod()
	log.Printf("Random pod: %v", pod)

	npart := strings.Split(pod.Name, "-")
	newPrefix := fmt.Sprintf("%v-%x", npart[0], rand.Intn(1<<16))
	oldPrefix := fmt.Sprintf("%v-%v", npart[0], npart[1])
	replicas := 0
	var srcPod *PodDetails

	// down
	for _, status := range PodStatusSequenceStop {
		seq := 0
		for {
			podName := fmt.Sprintf("%v-%x", oldPrefix, seq)
			oldPod, ok := view.Pods[pod.Namespace+"/"+podName]
			if !ok {
				replicas = seq
				break
			}
			srcPod = oldPod
			view.updateStatus(oldPod, []int{status})
			seq++
		}
	}

	// up
	for _, status := range PodStatusSequenceStart {
		for seq := 0; seq < replicas; seq++ {
			podName := fmt.Sprintf("%v-%x", newPrefix, seq)
			newPod, ok := view.Pods[pod.Namespace+"/"+podName]
			if !ok {
				newPod = view.RecreatePod(newPrefix, seq, srcPod)
			}
			view.updateStatus(newPod, []int{status})
		}
	}

}

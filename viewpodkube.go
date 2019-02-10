package main

import (
	"log"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"
)

// ViewPodKube extend ViewPod to represent pods from K8s
type ViewPodKube struct {
	ViewPod
	clientset *kubernetes.Clientset
}

// NewViewPodKube inits ViewPodKube struct
func NewViewPodKube(app KubexyApp) *ViewPodKube {
	log.Printf("Initialize ViewPod")

	config, err := clientcmd.BuildConfigFromFlags("", app.Args.KubeConfig)
	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	view := ViewPodKube{}
	view.Pods = make(map[string]*PodDetails)
	view.clientset = clientset

	view.refreshList()

	go view.watch()

	return &view
}

func (view *ViewPodKube) refreshList() error {

	// load data from k8s
	pods, err := view.clientset.CoreV1().Pods("").List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	// get current list of pods to detect removed
	podsOld := make(map[string]*PodDetails)
	for pod, podDet := range view.Pods {
		podsOld[pod] = podDet
	}

	view.mutex.Lock()
	defer view.mutex.Unlock()

	for _, pod := range pods.Items {
		newPod := view.createPod(&pod)
		if oldPod, ok := view.Pods[newPod.GetFullName()]; ok {
			if !oldPod.Equal(newPod) {
				log.Printf("update pod %v -> %v", oldPod, newPod)
				oldPod.Status = newPod.Status
			}
		} else {
			log.Printf("add pod %v", newPod)
			view.Pods[newPod.GetFullName()] = newPod
		}
		delete(podsOld, newPod.GetFullName())
	}

	for pod := range podsOld {
		log.Printf("delete pod %v", view.Pods[pod])
		delete(view.Pods, pod)
	}

	return nil
}

func (view *ViewPodKube) watch() {

	// FIXME
	for {
		time.Sleep(time.Duration(5) * time.Second)
		view.refreshList()
	}

}

func (view *ViewPodKube) createPod(pod *corev1.Pod) *PodDetails {

	// owner
	ownerKind := "undefined"
	if len(pod.ObjectMeta.OwnerReferences) > 0 {
		ownerKind = pod.ObjectMeta.OwnerReferences[0].Kind
	}

	// size
	var size int64
	for _, container := range pod.Spec.Containers {
		cSize := container.Resources.Requests.Cpu().MilliValue() * container.Resources.Requests.Memory().Value()
		if cSize == 0 {
			cSize = container.Resources.Limits.Cpu().MilliValue() * container.Resources.Limits.Memory().Value()
		}
		size += cSize
	}

	// status
	status := view.getPodStatus(pod)

	return &PodDetails{
		Name:      pod.ObjectMeta.Name,
		Namespace: pod.ObjectMeta.Namespace,
		OwnerKind: ownerKind,
		Node:      pod.Spec.NodeName,
		Status:    status,
		Size:      int(size / 1000 / 1024 / 1024)}

}

func (view *ViewPodKube) getPodStatus(pod *corev1.Pod) int {

	// status based on pod phase
	switch pod.Status.Phase {

	case corev1.PodFailed, corev1.PodUnknown:
		return PodStatusError

	case corev1.PodSucceeded:
		return PodStatusCompleted

	case corev1.PodPending:

		if pod.Spec.NodeName != "" {
			return PodStatusStarting
		}
		return PodStatusInit

	case corev1.PodRunning:
		// noop

	default:
		return PodStatusError

	}

	// terminating
	if pod.ObjectMeta.DeletionTimestamp != nil {
		return PodStatusTerminating
	}

	// check containers
	notReady := len(pod.Spec.Containers)
	for _, container := range pod.Status.ContainerStatuses {
		if container.Ready {
			notReady--
		} else {
			if container.State.Terminated != nil {
				return PodStatusCrash
			}
		}
	}

	if notReady > 0 {
		return PodStatusStarting
	}

	return PodStatusRunning
}

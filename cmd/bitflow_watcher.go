package cmd

import (
	"k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"strings"
	"time"
)

type BitflowWatcher struct {
	svc *BitflowService
	pods      []*v1.Pod
}

func NewBitflowWatcher(svc *BitflowService) *BitflowWatcher {
	return &BitflowWatcher{
		svc: svc,
	}
}

func (watcher *BitflowWatcher) WaitForReadyPods() error {

	err := watcher.SyncPods()
	if err != nil {
		return err
	}
	for {
		nodes, err := watcher.svc.Client().CoreV1().Nodes().List(v12.ListOptions{})
		if err != nil {
			return err
		}
		wellNodes := 0
		for _, node := range nodes.Items {
			nodePod, err := watcher.svc.GetBitflowPodForNode(node.Name)
			if err != nil {
				continue
			}
			if nodePod == nil || nodePod.Status.Phase != "Running" {
				continue
			}
			wellNodes++
		}

		if wellNodes == len(nodes.Items) {
			break
		}

		time.Sleep(1 * time.Second)
	}

	return nil
}

func (watcher *BitflowWatcher) WatchSync(interval time.Duration) {
	for {
		time.Sleep(interval)
		watcher.SyncPods()
	}
}

func (watcher *BitflowWatcher) SyncPods() error {
	// get nodes

	nodes, err := watcher.svc.Client().CoreV1().Nodes().List(v12.ListOptions{})
	if err != nil {
		return err
	}
	for _, node := range nodes.Items {
		log.Printf("looking for pod %s", node.Name)
		nodePod, err := watcher.svc.GetBitflowPodForNode(node.Name)

		if err != nil {
			if strings.Contains(err.Error(), "not found"){
				nodePod, err = watcher.svc.CreateBitflowPod(node.Name)

				if err != nil {
					return err
				}

				log.Printf("created pod %s", nodePod.Name)
				watcher.pods = append(watcher.pods, nodePod)
			} else {
				return err
			}
		} else {
			log.Printf("pod %s running", nodePod.Name)
		}
	}

	return nil
}
func (watcher *BitflowWatcher) DeletePods() error {
	// get nodes

	nodes, err := watcher.svc.Client().CoreV1().Nodes().List(v12.ListOptions{})
	if err != nil {
		return err
	}
	for _, node := range nodes.Items {
		nodePod, err := watcher.svc.GetBitflowPodForNode(node.Name)
		if err == nil {
			log.Printf("deleting pod %s...", nodePod.Name)
			err = watcher.svc.Client().CoreV1().Pods(nodePod.Namespace).Delete(nodePod.Name, &v12.DeleteOptions{})

			if err != nil {
				return err
			}
			log.Printf("pod %s deleted", nodePod.Name)
		}
	}

	return nil
}


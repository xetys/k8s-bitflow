package cmd

import (
	"bytes"
	"io"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	"log"
	"math/rand"
	"strings"
	"time"
)

type Updater struct {
	svc       *BitflowService
	timeoutId int
	working   bool
}

func NewUpdater(svc *BitflowService) *Updater {
	return &Updater{
		svc: svc,
	}
}

func (updater *Updater) ScheduleBitflowUpdate() {
	id := rand.Int()
	updater.timeoutId = id
	updater.wait()
	go func(id int) {
		time.Sleep(5 * time.Second)

		if id == updater.timeoutId {
			updater.updateBitflowMetrics()
		}
	}(id)
}

func (updater *Updater) wait() {
	for updater.working {
		time.Sleep(100)
	}
}

func (updater *Updater) updateBitflowMetrics() error {
	updater.working = true
	log.Println("starting bitflow reconfiguration...")
	nodes, err := updater.svc.Client().CoreV1().Nodes().List(v1.ListOptions{})
	if err != nil {
		updater.working = false
		return err
	}

	log.Println("clear old procs")
	for _, node := range nodes.Items {
		// delete old procs
		err = updater.runOnNode(node.Name, "api/proc-children", "DELETE")

		if err != nil {
			updater.working = false
			return err
		}
	}

	podList, err := updater.svc.Client().CoreV1().Pods("").List(v1.ListOptions{})

	if err != nil {
		updater.working = false
		return err
	}

	log.Printf("setup %d new procs\n", len(podList.Items))
	for _, pod := range podList.Items {
		podName := pod.Name
		containerStatus := pod.Status.ContainerStatuses[0]
		containerSlice := containerStatus.ContainerID
		if len(containerSlice) < 10 {
			log.Printf("cloud not read container id from %s, skipping\n", containerSlice)
			continue
		}
		containerId := containerSlice[9:]

		err = updater.runOnNode(pod.Spec.NodeName, "api/proc-children/"+podName+"?regex="+containerId, "POST")
		if err != nil {
			updater.working = false
			return err
		}
	}
	log.Println("done")
	updater.working = false
	return nil
}

func (updater *Updater) runOnNode(nodeName string, url string, method string) error {
	command := "curl localhost:7777/" + url
	if method > "" {
		command += " -X" + method
	}
	pod, err := updater.svc.GetBitflowPodForNode(nodeName)

	if err != nil {
		return err
	}

	execRequest := updater.svc.Client().CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("exec")

	execRequest.VersionedParams(&v12.PodExecOptions{
		Container: "bitflow",
		Command:   strings.Split(command, " "),
		Stderr:    true,
		Stdout:    true,
	}, scheme.ParameterCodec)

	config, err := K8SConfig()
	exec, err := remotecommand.NewSPDYExecutor(config, "POST", execRequest.URL())

	if err != nil {
		return err
	}

	var (
		execOut bytes.Buffer
		execErr bytes.Buffer
	)

	err = exec.Stream(remotecommand.StreamOptions{
		Stdout: &execOut,
		Stderr: &execErr,
		Tty:    false,
	})

	if err != nil {
		log.Println(execOut.String(), execErr.String())
		return err
	}

	return nil
}

type Writer struct {
	Str []string
}

func (w *Writer) Write(p []byte) (n int, err error) {
	str := string(p)
	if len(str) > 0 {
		w.Str = append(w.Str, str)
	}
	return len(str), nil
}
func newStringReader(ss []string) io.Reader {
	formattedString := strings.Join(ss, "\n")
	reader := strings.NewReader(formattedString)
	return reader
}

package cmd

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type BitflowService struct {
	clientset *kubernetes.Clientset
}

func (svc *BitflowService) Client() *kubernetes.Clientset {
	return svc.clientset
}

func NewBitflowService(clientset *kubernetes.Clientset) *BitflowService {
	return &BitflowService{
		clientset: clientset,
	}
}
func (svc *BitflowService) GetBitflowPodForNode(nodeName string) (*v1.Pod, error) {
	podName := "bitflow-" + nodeName
	return svc.clientset.CoreV1().Pods(v1.NamespaceDefault).Get(podName, v12.GetOptions{})
}

func (svc *BitflowService) CreateBitflowPod(nodeName string) (*v1.Pod, error) {
	podName := "bitflow-" + nodeName
	selector := make(map[string]string)
	selector["kubernetes.io/hostname"] = nodeName
	tr := true
	pod := &v1.Pod{
		ObjectMeta: v12.ObjectMeta{
			Name: podName,
			Labels: map[string]string{
				"collector-name": podName,
			},
		},
		Spec: v1.PodSpec{
			HostNetwork:        true,
			HostPID:            true,
			ServiceAccountName: "bitflow",
			Tolerations: []v1.Toleration{
				{
					Effect:   v1.TaintEffectNoSchedule,
					Key:      "node-role.kubernetes.io/master",
					Operator: v1.TolerationOpExists,
				},
			},
			Containers: []v1.Container{
				{
					Name:            "bitflow",
					Image:           "xetys/bitflow-collector",
					ImagePullPolicy: v1.PullAlways,
					Command: []string{
						"./bitflow-collector",
						"-ci",
						"500ms",
						"-api",
						":7777",
						"-o",
						":5010",
						"-o",
						"http+prometheus://:5090",
					},
					Ports: []v1.ContainerPort{
						{Name: "web", ContainerPort: 7777},
						{Name: "tcp", ContainerPort: 5010},
						{Name: "prometheus", ContainerPort: 5090},
					},
					SecurityContext: &v1.SecurityContext{
						Privileged: &tr,
					},
				},
			},
			NodeSelector: selector,
		},
	}

	pod, err := svc.clientset.CoreV1().Pods(v1.NamespaceDefault).Create(pod)

	if err != nil {
		return nil, err
	}

	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: podName,
			Labels: map[string]string{
				"bitflow-component": "collector",
			},
		},
		Spec: v1.ServiceSpec{
			Selector: map[string]string{
				"collector-name": podName,
			},
			Ports: []v1.ServicePort{
				{
					Name: "prometheus",
					Port: 5090,
				},
			},
		},
	}

	service, err = svc.clientset.CoreV1().Services(v1.NamespaceDefault).Create(service)
	if err != nil {
		return nil, err
	}

	return pod, nil
}

package main

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"os"
	"path/filepath"
)

type KubeClient struct {
	clientset *kubernetes.Clientset
	kubeconfig *string
}

func NewKubeConfig (ctx context.Context) (*string, error) {
	if home := homedir.HomeDir(); home != "" {
		kc := filepath.Join(home, ".kube", "config")
		if _, err := os.Stat(kc) ; err != nil {
			return nil, err
		} else {
			return &kc, nil
		}
	} else {
		return nil, errors.New("kube config path error")
	}
}

func (kc *KubeClient) Clientset() *kubernetes.Clientset{
	return kc.clientset
}

func NewKubeClient (kubeconfig *string) (*KubeClient, error) {
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &KubeClient{
		clientset: clientset,
		kubeconfig: kubeconfig,
	}, nil
}

type Watcher interface {
	Status() int8
	Watch() error
	Stop() error
}

type PodWatcher struct {
}

func main(){
	sub2()

}

func sub2() {
	conf, err := NewKubeConfig(context.Background())
	if err != nil {
		logrus.Panic(err)
	}
	kc, err := NewKubeClient(conf)
	if err != nil {
		logrus.Panic(err)
	}

	crd := NewClusterResourceDescriber(kc)
	nodeSelector, err := nodeParseSelector("")
	if err != nil {
		panic(err)
	}
	pods, err := crd.getPodsList(context.TODO(),"default", nodeSelector)
	if err != nil {
		panic(err)
	}
	for _, pod := range pods.Items {
		fmt.Println(pod.Name,pod.Spec.NodeName)
	}

}

func sub () {
	conf, err := NewKubeConfig(context.Background())
	if err != nil {
		logrus.Panic(err)
	}
	kc, err := NewKubeClient(conf)
	if err != nil {
		logrus.Panic(err)
	}

	api := kc.Clientset().CoreV1()
	// kc.Clientset().CoreV1()
	nodes, err := api.Nodes().List(context.TODO(), metav1.ListOptions{})
	fmt.Println(len(nodes.Items))
	gpuAsRM := corev1.ResourceName("custom.com/dongle")
	for _, e := range nodes.Items {
		q := e.Status.Allocatable[gpuAsRM]
		if numAsInt64, ok := q.AsInt64(); !ok {
		} else {
			fmt.Println("abcded",numAsInt64)
		}
	}
	watcher, err := api.Pods("dev").Watch(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Println(err)
	}
	ch := watcher.ResultChan()
	for event := range ch {
		podEvent, ok := event.Object.(*corev1.Pod)
		if ok {
			// fmt.Println(podEvent.String())
			fmt.Println(podEvent.UID, podEvent.Status.Phase)
			for _, pc := range podEvent.Status.Conditions {
				fmt.Printf(".......%s\n",pc)
			}
		}
	}

}
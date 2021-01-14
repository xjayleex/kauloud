package k8s

import (
	"github.com/pkg/errors"
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

func NewKubeConfig () (*string, error) {
	// TODO :: 1. make to build kube config from cli and yaml configuration.

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

/*
func sub2() {
	clientConfig := kubecli.DefaultClientConfig(&pflag.FlagSet{})
	virtClient, err := kubecli.GetKubevirtClientFromClientConfig(clientConfig)
	if err != nil {
		panic(err)
	}
	vmList, err := virtClient.VirtualMachine("default").List(&metav1.ListOptions{})
	if err != nil {
		panic(err)
	}
	for _, vm := range vmList.Items {
		fmt.Println(vm.Name)
	}
}

func sub () {
	conf, err := NewKubeConfig()
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

	watcher, err := api.Pods("default").Watch(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Println(err)
	}

	ch := watcher.ResultChan()
	for event := range ch {
		podEvent, ok := event.Object.(*v1.Pod)
		if ok {
			// fmt.Println(podEvent.String())
			fmt.Println(podEvent.UID, podEvent.Status.Phase, podEvent.Status.PodIP)
			fmt.Println("Req : ", podEvent.Spec.Containers[0].Resources.Requests)
			fmt.Println("Limit : ", podEvent.Spec.Containers[0].Resources.Limits)
			//podRequestsAndLimits(podEvent) // 이거 될까 ?
			for _, pc := range podEvent.Status.Conditions {
				fmt.Printf(".......%s\n",pc)
			}
		}
	}
}
*/
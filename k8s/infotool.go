package k8s

import (
	"errors"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	clientset "k8s.io/client-go/kubernetes"
)

type ClusterResourceDescriber struct {
	nodeLists   *corev1.NodeList
	cliset      *clientset.Clientset
	reqs        map[corev1.ResourceName]resource.Quantity
	limits      map[corev1.ResourceName]resource.Quantity
	allocatable map[corev1.ResourceName]resource.Quantity
}

func NewClusterResourceDescriber (kcli *KubeClient) *ClusterResourceDescriber {
	return &ClusterResourceDescriber{
		cliset: kcli.Clientset(),
	}
}

func (crd *ClusterResourceDescriber) UpdatedNodeList (ctx context.Context) (*corev1.NodeList, error) {
	if crd.cliset == nil {
		return nil, errors.New("cliset nil error")
	}

	nodes, err := crd.cliset.CoreV1().Nodes().List(ctx,  metav1.ListOptions{})
	return nodes, err
}

func (crd *ClusterResourceDescriber) NodeList() (*corev1.NodeList, error) {
	if crd.nodeLists == nil {
		updated, err := crd.UpdatedNodeList(context.Background())
		if err != nil {
			return nil, err
		} else {
			crd.nodeLists = updated
		}
	}
	return crd.nodeLists, nil
}

func (crd *ClusterResourceDescriber) UpdatedAllocatable() (corev1.ResourceList, error) {
	if crd.allocatable == nil {
		crd.allocatable = corev1.ResourceList{}
	}

	nodeLists, err := crd.NodeList()

	if err != nil {
		return nil, err
	}

	for _, node := range nodeLists.Items {
		addResourceList(crd.allocatable, node.Status.Allocatable)
	}

	return crd.allocatable, nil
}

func (crd *ClusterResourceDescriber) Allocatable() (corev1.ResourceList, error) {
	if crd.allocatable == nil {
		return nil, errors.New("Nil Allocatable error")
	}
	return crd.allocatable, nil
}

// Always makes network requests.
func (crd *ClusterResourceDescriber) getPodsList (ctx context.Context, namespace string, selector fields.Selector) (*PodsList, error) {
	podsList, err := crd.cliset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{FieldSelector: selector.String()})
	if err != nil {
		return nil, err
	}
	return &PodsList{podsList} , err
}

func (crd *ClusterResourceDescriber) DescribeAllNode (ctx context.Context, namespace string) (interface{}, error) {
	allSelector, err := nodeParseSelector("")
	if err != nil {
		return nil, err
	}

	podsList, err := crd.getPodsList(ctx, namespace, allSelector)
	if err != nil {
		return nil, err
	}
	crd.reqs, crd.limits = podsList.describe()
}

type PodsList struct {
	*corev1.PodList
}

func (podsList *PodsList) describe () (reqs, limits corev1.ResourceList){
	reqs, limits = corev1.ResourceList{}, corev1.ResourceList{}
	for _, pod := range podsList.Items {
		podReqs, podLimits := podRequestsAndLimits(&pod)
		for podReqName, podReqValue := range podReqs {
			if value, ok := reqs[podReqName]; !ok {
				reqs[podReqName] = podReqValue.DeepCopy()
			} else {
				value.Add(podReqValue)
				reqs[podReqName] = value
			}
		}
		for podLimitName, podLimitValue := range podLimits {
			if value, ok := limits[podLimitName]; !ok {
				reqs[podLimitName] = podLimitValue.DeepCopy()
			} else {
				value.Add(podLimitValue)
				reqs[podLimitName] = value
			}
		}
	}
	return reqs, limits
}

// if node == "" , return all node data.
func nodeParseSelector (node string) (fields.Selector, error){
	if node == "" {
		return fields.ParseSelector("status.phase!=" + string(corev1.PodSucceeded) + ",status.phase!=" + string(corev1.PodFailed))
	} else {
		return fields.ParseSelector("spec.nodeName=" + node + ",status.phase!=" + string(corev1.PodSucceeded) + ",status.phase!=" + string(corev1.PodFailed))
	}
}

func podRequestsAndLimits(pod *corev1.Pod) (reqs, limits corev1.ResourceList) {
	reqs, limits = corev1.ResourceList{}, corev1.ResourceList{}
	for _, container := range pod.Spec.Containers {
		addResourceList(reqs, container.Resources.Requests)
		addResourceList(limits, container.Resources.Limits)
	}
	// init containers define the minimum of any resource
	for _, container := range pod.Spec.InitContainers {
		maxResourceList(reqs, container.Resources.Requests)
		maxResourceList(limits, container.Resources.Limits)
	}

	// Add overhead for running a pod to the sum of requests and to non-zero limits:
	if pod.Spec.Overhead != nil {
		addResourceList(reqs, pod.Spec.Overhead)

		for name, quantity := range pod.Spec.Overhead {
			if value, ok := limits[name]; ok && !value.IsZero() {
				value.Add(quantity)
				limits[name] = value
			}
		}
	}
	return
}

func addResourceList(list, new corev1.ResourceList) {
	for name, quantity := range new {
		if value, ok := list[name]; !ok {
			list[name] = quantity.DeepCopy()
		} else {
			value.Add(quantity)
			list[name] = value
		}
	}
}

func maxResourceList(list, new corev1.ResourceList) {
	for name, quantity := range new {
		if value, ok := list[name]; !ok {
			list[name] = quantity.DeepCopy()
			continue
		} else {
			if quantity.Cmp(value) > 0 {
				list[name] = quantity.DeepCopy()
			}
		}
	}
}
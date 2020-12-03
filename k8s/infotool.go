package main

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
	clientset   *clientset.Clientset
	reqs        map[corev1.ResourceName]resource.Quantity
	limits      map[corev1.ResourceName]resource.Quantity
	allocatable map[corev1.ResourceName]resource.Quantity

	r			ResourceMap
	l			ResourceMap
	a			ResourceMap
}
type ResourceMap struct {
	PerNode map[string]map[corev1.ResourceName]resource.Quantity
	Total   map[corev1.ResourceName]resource.Quantity
}

func NewResourceMap (nodes *corev1.NodeList) *ResourceMap {
	perNode := make(map[string]map[corev1.ResourceName]resource.Quantity)
	for _, node := range nodes.Items {
		perNode[node.Name] = make(map[corev1.ResourceName]resource.Quantity)
	}
	total := make(map[corev1.ResourceName]resource.Quantity)
	return &ResourceMap{
		PerNode: perNode,
		Total:   total,
	}
}

func (rm *ResourceMap) Add() {
	// rm.Pernode에
}
func (rm *ResourceMap) Sub() {}
func (rm *ResourceMap) RefByKey() {}

func NewClusterResourceDescriber (client *KubeClient) *ClusterResourceDescriber {
	return &ClusterResourceDescriber{
		clientset: client.Clientset(),
	}
}

func (crd *ClusterResourceDescriber) UpdatedNodeList (ctx context.Context) (*corev1.NodeList, error) {
	if crd.clientset == nil {
		return nil, errors.New("clientset nil error")
	}

	nodes, err := crd.clientset.CoreV1().Nodes().List(ctx,  metav1.ListOptions{})
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
	podsList, err := crd.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{FieldSelector: selector.String()})
	if err != nil {
		return nil, err
	}
	return &PodsList{podsList} , err
}
//
//
// Describe each node
// 		nodeA  nodeB  nodeC
// cpu  1000m  2000m  2000m
// gpu    1      0      2
// mem
//        \      |      /
//         \     |     /
//		    \    |    /
//           \   |   /
//            \  |  /
//    map describing all node
// cpu           5
// gpu           3
// mem ~~~~~~~~~~~~~~~~~~~~

func (crd *ClusterResourceDescriber) DescribeAllNodes (ctx context.Context, namespace string) (interface{}, error) {
	allSelector, err := nodeParseSelector("")
	if err != nil {
		return nil, err
	}

	podsList, err := crd.getPodsList(ctx, namespace, allSelector)
	if err != nil {
		return nil, err
	}

	// for each node -> compute Allocatable
	nodes, err := crd.NodeList()
	if err != nil {
		return nil, err
	}
	for _, node := range nodes.Items {

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
package main

import (
	"errors"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
)

type AllocationGRPC struct {
	Server *grpc.Server
	*ClusterResourceDescriber
}

func NewAllocationGRPC (gs *grpc.Server, crd *ClusterResourceDescriber) *AllocationGRPC {
	return &AllocationGRPC{
		Server: gs,
		ClusterResourceDescriber: crd,
	}
}

type ClusterResourceDescriber struct {
	namespace  string
	nodeLists   *v1.NodeList
	clientset   *kubernetes.Clientset

	requested   *ResourceMap
	limit       *ResourceMap
	allocatable *ResourceMap
}

func NewClusterResourceDescriber (client *KubeClient, namespace string) (*ClusterResourceDescriber, error) {
	// TODO :: make `namespace` on the cmd codes.
	crd := &ClusterResourceDescriber{clientset: client.Clientset()}
	crd.namespace = namespace
	_, err := crd.NodeList()
	if err != nil {
		return nil, err
	}

	_, err = crd.UpdatedAllocatable(context.Background())
	if err != nil {
		return nil, err
	}
	_, _, err = crd.DescribeAllNodes(context.Background())
	if err != nil {
		return nil, err
	}

	return crd, nil
}


func (crd *ClusterResourceDescriber) UpdatedNodeList (ctx context.Context) (*v1.NodeList, error) {
	if crd.clientset == nil {
		return nil, errors.New("clientset nil error")
	}

	nodes, err := crd.clientset.CoreV1().Nodes().List(ctx,  metav1.ListOptions{})
	return nodes, err
}

func (crd *ClusterResourceDescriber) NodeList() (*v1.NodeList, error) {
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

func (crd *ClusterResourceDescriber) UpdatedAllocatable(ctx context.Context) (*ResourceMap, error) {
	nodeList, err := crd.UpdatedNodeList(ctx)
	if err != nil {
		return nil, err
	}
	crd.allocatable = NewResourceMap(nodeList)

	for _, node := range nodeList.Items {
		crd.allocatable.Assign(node.Name, node.Status.Allocatable)
	}

	return crd.allocatable, nil
}

func (crd *ClusterResourceDescriber) Allocatable() (*ResourceMap, error) {
	if crd.allocatable == nil {
		updated, err := crd.UpdatedAllocatable(context.Background())
		if err != nil {
			return nil, err
		} else {
			crd.allocatable = updated
		}
	}
	return crd.allocatable, nil
}

func (crd *ClusterResourceDescriber) Requested () *ResourceMap {
	return crd.requested
}

func (crd *ClusterResourceDescriber) Limit () *ResourceMap {
	return crd.limit
}

// Always makes network requested.
func (crd *ClusterResourceDescriber) getPodsList (ctx context.Context, selector fields.Selector) (*PodsList, error) {
	podsList, err := crd.clientset.CoreV1().Pods(crd.namespace).List(ctx, metav1.ListOptions{FieldSelector: selector.String()})
	if err != nil {
		return nil, err
	}
	return &PodsList{podsList} , err
}

// Describe each node
//      nodeA  nodeB  nodeC
// cpu  1000m  2000m  2000m
// gpu    1      0      2
// mem
//        \      |      /
//         \     |     /
//          \    |    /
//           \   |   /
//            \  |  /
//    map describing all node
// cpu           5
// gpu           3
// mem ~~~~~~~~~~~~~~~~~~~~

func (crd *ClusterResourceDescriber) DescribeAllNodes (ctx context.Context) (*ResourceMap, *ResourceMap, error) {
	allSelector, err := nodeParseSelector("")
	if err != nil {
		return nil, nil, err
	}

	podsList, err := crd.getPodsList(ctx, allSelector)
	if err != nil {
		return nil, nil, err
	}
	nodeList, err := crd.NodeList()
	if err != nil {
		return nil, nil, err
	}

	crd.requested, crd.limit = NewResourceMap(nodeList), NewResourceMap(nodeList)
	for _, node := range nodeList.Items {
		req, limit := podsList.describe(node.Name)
		crd.requested.Assign(node.Name, req)
		crd.limit.Assign(node.Name, limit)
	}

	return crd.requested, crd.limit, nil
}


type ResourceMap struct {
	PerNode map[string]v1.ResourceList
	Total   v1.ResourceList

}

func NewResourceMap (nodes *v1.NodeList) *ResourceMap {
	perNode := make(map[string]v1.ResourceList)
	for _, node := range nodes.Items {
		perNode[node.Name] = make(v1.ResourceList)
	}
	total := make(v1.ResourceList)
	return &ResourceMap{
		PerNode: perNode,
		Total:   total,
	}
}

// Assign New
func (rm *ResourceMap) Assign(node string, resources v1.ResourceList) {
	rm.PerNode[node] = resources

	for resourceName, resourceValue := range resources {
		if value , ok := rm.Total[resourceName]; !ok {
			rm.Total[resourceName] = resourceValue.DeepCopy()
		} else {
			value.Add(resourceValue)
			rm.Total[resourceName] = value
		}
	}
}

func (rm *ResourceMap) Add(node string, resources v1.ResourceList) {
	for resourceName, resourceValue := range resources {
		value, _ := rm.PerNode[node][resourceName] // 가지고 있던 quantity
		value.Add(resourceValue)
		rm.PerNode[node][resourceName] = value

		value, _ = rm.Total[resourceName]
		value.Add(resourceValue)
		rm.Total[resourceName] = value
	}
}

func (rm *ResourceMap) Sub(node string, resources v1.ResourceList) {
	for resourceName, resourceValue := range resources {
		value, _ := rm.PerNode[node][resourceName] // 가지고 있던 quantity
		value.Sub(resourceValue)
		rm.PerNode[node][resourceName] = value

		value, _ = rm.Total[resourceName]
		value.Sub(resourceValue)
		rm.Total[resourceName] = value
	}
}

func (rm *ResourceMap) RefByKey() {}


type PodsList struct {
	*v1.PodList
}

func (podsList *PodsList) describe (node string) (reqs, limits v1.ResourceList){
	reqs, limits = v1.ResourceList{}, v1.ResourceList{}
	allFlag := false
	if node == "" {
		allFlag = true
	}
	for _, pod := range podsList.Items {
		if !allFlag && pod.Spec.NodeName != node {
			// node specific 정보를 얻으려하고, 명시한 node가 pod NodeName과 다르면 건너뜀.
			continue
		}
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
				limits[podLimitName] = podLimitValue.DeepCopy()
			} else {
				value.Add(podLimitValue)
				limits[podLimitName] = value
			}
		}
	}
	return reqs, limits
}

// if node == "" , return all node data.
func nodeParseSelector (node string) (fields.Selector, error){
	if node == "" {
		return fields.ParseSelector("status.phase!=" + string(v1.PodSucceeded) + ",status.phase!=" + string(v1.PodFailed))
	} else {
		return fields.ParseSelector("spec.nodeName=" + node + ",status.phase!=" + string(v1.PodSucceeded) + ",status.phase!=" + string(v1.PodFailed))
	}
}

func podRequestsAndLimits(pod *v1.Pod) (reqs, limits v1.ResourceList) {
	reqs, limits = v1.ResourceList{}, v1.ResourceList{}
	for _, container := range pod.Spec.Containers {
		addResourceList(reqs, container.Resources.Requests)
		addResourceList(limits, container.Resources.Limits)
	}
	// init containers define the minimum of any resource
	for _, container := range pod.Spec.InitContainers {
		maxResourceList(reqs, container.Resources.Requests)
		maxResourceList(limits, container.Resources.Limits)
	}

	// Add overhead for running a pod to the sum of requested and to non-zero limits:
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

func addResourceList(list, new v1.ResourceList) {
	for name, quantity := range new {
		if value, ok := list[name]; !ok {
			list[name] = quantity.DeepCopy()
		} else {
			value.Add(quantity)
			list[name] = value
		}
	}
}

func maxResourceList(list, new v1.ResourceList) {
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
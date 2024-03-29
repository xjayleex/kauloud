package informer

import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/xjayleex/kauloud/pkg/api/kauloud"
	"github.com/xjayleex/kauloud/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"time"
)

type ResourceDescriberInterface interface {
	IsAllocatable(requests corev1.ResourceList) bool
	IsRunning() bool
	Run(int, chan struct{})
}

type StreamingResourceDescriber struct {
	config			*utils.KauloudConfig
	clientset 		*kubernetes.Clientset
	logger			*logrus.Logger
	running			bool

	requests   		*ResourceMap
	limits     		*ResourceMap
	allocatable	 	*ResourceMap

	nodeList 		map[string]*corev1.Node
	podList			map[string]*corev1.Pod

	podInformer		cache.SharedInformer
	nodeInformer	cache.SharedInformer

	Queue			workqueue.RateLimitingInterface
}

func NewStreamingResourceDescriber(config *utils.KauloudConfig, clientset *kubernetes.Clientset, logger *logrus.Logger) *StreamingResourceDescriber {
	new := &StreamingResourceDescriber{
		config: 	 config,
		clientset:   clientset,
		logger:      logger,
		running: false,
		requests:    NewResourceMap(),
		limits:      NewResourceMap(),
		allocatable: NewResourceMap(),
		nodeList:    make(map[string]*corev1.Node),
		podList:     make(map[string]*corev1.Pod),
		Queue:       workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
	}

	podLW := cache.NewListWatchFromClient(new.clientset.CoreV1().RESTClient(),"pods", kauloud.AllNamespace, fields.Everything() )
	new.podInformer = cache.NewSharedInformer(podLW, &corev1.Pod{}, 0)
	new.podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    new.addPod,
		UpdateFunc: new.updatePod,
		DeleteFunc: new.deletePod,
	})

	nodeLW := cache.NewListWatchFromClient(new.clientset.CoreV1().RESTClient(), "nodes", kauloud.AllNamespace, fields.Everything())
	new.nodeInformer = cache.NewSharedInformer(nodeLW, &corev1.Node{}, 0)
	new.nodeInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: new.addNode,
		UpdateFunc: new.updateNode,
		DeleteFunc: new.deleteNode,
	})
	return new
}

func (o *StreamingResourceDescriber) CheckAvailability(requests corev1.ResourceList) bool {
	for nodeName, _ := range o.nodeList {
		// if expectedAfterAlloc << o.allocatable -> return true
		// Todo or not :: node condition Ready Check
		if requested, err := o.requests.RefNodeResourceList(nodeName); err != nil {
			continue
		} else {
			if expected := expectedAfterAllocation(requested, requests); o.isExpectedEnough(nodeName, expected) {
				return true
			} else {
				continue
			}
		}
	}
	return false
}

func (o *StreamingResourceDescriber) isExpectedEnough (nodeName string, expected corev1.ResourceList) bool {
	if o.nodeList[nodeName] == nil {
		return false
	}
	nodeAllocatable, err := o.allocatable.RefNodeResourceList(nodeName)
	if err != nil {
		return false
	}

	// `cpu` is compressible resource. We can avoid checking cpu overcommit.
	// Reference. https://github.com/kubernetes/community/blob/master/contributors/design-proposals/node/resource-qos.md#incompressible-resource-guarantees
	if o.config.ResourceDescriber.SkipCompressibleResource {
		delete(expected, corev1.ResourceName("cpu"))
	}

	for resourceName, quantity := range expected {
		allocatableQuantity, ok := nodeAllocatable[resourceName]
		if !ok {
			return false
		}
		if allocatableQuantity.Cmp(quantity) == -1 {
			return false
		} else { continue }
	}
	return true
}

func (o *StreamingResourceDescriber) Run(threadiness int, stopCh chan struct{}) {
	defer runtime.HandleCrash()
	defer o.Queue.ShutDown()
	o.logger.Infoln("Starting Cluster Resource Describer.")
	go o.podInformer.Run(stopCh)
	go o.runTestLister(stopCh)
	// go o.nodeInformer.Run(stopCh)

	if !cache.WaitForCacheSync(stopCh, o.podInformer.HasSynced) {
		runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}

	for i := 0 ; i < threadiness; i++ {
		o.logger.Info("Run watch dogs for polling resource describer.")
		go wait.Until(o.runWorker, time.Second, stopCh)
	}
	o.running = true
	defer func() {o.running = false}()
	<-stopCh
	o.logger.Infoln("Stopping describer.")
}

func (o *StreamingResourceDescriber) IsRunning() bool {
	return o.running
}

func (o *StreamingResourceDescriber) runWorker() {
	for o.consume() {}
}

func (o *StreamingResourceDescriber) consume() bool {
	key, quit := o.Queue.Get()
	if quit {
		return false
	}
	defer o.Queue.Done(key)

	err := o.sync(key.(Event))
	o.handleError(err, key)
	return true
}

func (o *StreamingResourceDescriber) sync(event Event) error {
	var object interface{}
	var exists bool
	var err error

	switch event.resource {
	case kauloud.ResourceAbbrPod:
		object, exists, err = o.podInformer.GetStore().GetByKey(event.key.(string))
	case kauloud.ResourceAbbrNode:
		object, exists, err = o.nodeInformer.GetStore().GetByKey(event.key.(string))
	}

	if err != nil {
		o.logger.Infof("error fetching object from index for the specified key. %s %v", event.key.(string), err)
		return err
	}

	if !exists {
		err := o.deleteHandler(object, &event)
		return err
	}

	if event.verb == kauloud.WatcherEventVerbAdd {
		err = o.addHandler(object, &event)
	} else { // "update"
		err = o.updateHandler(object, &event)
	}

	return err
}

func (o *StreamingResourceDescriber) addHandler(object interface{}, event *Event) (err error) {
	o.logger.Debugf("Add received for %s", event.key.(string))
	switch event.resource {
	case kauloud.ResourceAbbrPod:
		err = o.handlePodAddition(object)
	case kauloud.ResourceAbbrNode:
		err = o.handleNodeAddition(object)
	}
	return err
}

func (o *StreamingResourceDescriber) deleteHandler(object interface{}, event *Event) (err error) {
	o.logger.Debugf("Delete received for %s", event.key.(string))
	switch event.resource {
	case kauloud.ResourceAbbrPod:
		err = o.handlePodDeletion(object, event)
	case kauloud.ResourceAbbrNode:
		err = o.handleNodeDeletion(object, event)
	}
	return err
}

func (o *StreamingResourceDescriber) updateHandler(object interface{}, event *Event) (err error) {
	o.logger.Debugf("Update received for %s", event.key.(string))
	switch event.resource {
	case kauloud.ResourceAbbrPod:
		err = o.handlePodUpdate(object)
	case kauloud.ResourceAbbrNode:
		o.handleNodeUpdate(object)
	}
	return err
}

func (o *StreamingResourceDescriber) handlePodAddition (object interface{}) error {
	pod, ok := object.(*corev1.Pod)
	if !ok {
		return kauloud.ErrorTypeAssertionForPod
	}
	podName := ConcatenatedPodNameWithNamespace(pod.Namespace, pod.Name)
	if pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodSucceeded  {
		return nil
	}

	o.podList[podName] = pod

	if pod.Status.Phase == corev1.PodPending && pod.Spec.NodeName == "" {
		return kauloud.ErrorPodPendingWithNoNode
	}

	requests, limits := PodRequestsAndLimits(pod)
	o.requests.add(pod.Spec.NodeName, requests)
	o.limits.add(pod.Spec.NodeName, limits)
	return nil
}

func (o *StreamingResourceDescriber) handlePodDeletion (object interface{}, event *Event) error {
	var podName string
	pod, ok := object.(*corev1.Pod)
	if ok {
		podName = ConcatenatedPodNameWithNamespace(pod.Namespace, pod.Name)
	} else {
		podName = event.key.(string)
		pod = o.podList[podName]
	}

	if _, exists := o.podList[podName]; !exists {
		return nil
	}

	requests, limits := PodRequestsAndLimits(pod)
	o.requests.sub(pod.Spec.NodeName, requests)
	o.limits.sub(pod.Spec.NodeName, limits)
	delete(o.podList, podName)

	return nil
}

func (o *StreamingResourceDescriber) handlePodUpdate (object interface{}) error {
	pod, ok := object.(*corev1.Pod)
	if !ok {
		return kauloud.ErrorTypeAssertionForNode
	}
	podName := ConcatenatedPodNameWithNamespace(pod.Namespace, pod.Name)

	if pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodSucceeded {
		if _, exists := o.podList[podName]; exists {
			requests, limits := PodRequestsAndLimits(pod)
			o.requests.sub(pod.Spec.NodeName, requests)
			o.limits.sub(pod.Spec.NodeName, limits)
			delete(o.podList, podName)
		}
		return nil
	}

	o.podList[podName] = pod
	return nil
}

// 1. Add Node object to the node list.
// 2. Add allocatable resource with the node.
func (o *StreamingResourceDescriber) handleNodeAddition (object interface{}) error {
	node, ok := object.(*corev1.Node)
	if !ok {
		return kauloud.ErrorTypeAssertionForNode
	}

	o.nodeList[node.Name] = node
	o.allocatable.add(node.Name, node.Status.Allocatable)
	return nil
}

// 1. Try to delete Node object from node list.
// 2. Try to delete allocatable resource associated with the node.
func (o *StreamingResourceDescriber) handleNodeDeletion (object interface{}, event *Event) error {
	var nodeName string
	node, ok := object.(*corev1.Node)
	if ok {
		nodeName = node.Name
	} else {
		nodeName = event.key.(string)
		if node, ok = o.nodeList[nodeName]; !ok {
			return errors.New("no node object on nodeList map")
		}
	}

	if _, exists := o.nodeList[nodeName]; !exists {
		return nil
	}

	delete(o.nodeList, nodeName)
	o.allocatable.deleteNode(nodeName)
	return nil
}

func (o *StreamingResourceDescriber) handleNodeUpdate (object interface{}) error {
	node, ok := object.(*corev1.Node)
	if !ok {
		return kauloud.ErrorTypeAssertionForNode
	}
	o.nodeList[node.Name] = node

	return nil
}

func (o *StreamingResourceDescriber) handleError (err error, key interface{}) {
	if err == nil {
		o.Queue.Forget(key)
		return
	}

	if o.Queue.NumRequeues(key) < kauloud.ControllerMaxRequeue {
		o.logger.Infof("error during sync %v %v", key, err)
		o.Queue.AddRateLimited(key)
		return
	}

	o.Queue.Forget(key)
	runtime.HandleError(err)
	o.logger.Infof("drop pod out of queue after many retries, %v %v", key, err)
}

func (o *StreamingResourceDescriber) addPod(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err == nil {
		event := Event{
			resource: kauloud.ResourceAbbrPod,
			verb: kauloud.WatcherEventVerbAdd,
			key: key,
		}
		o.Queue.Add(event)
	}
}

func (o *StreamingResourceDescriber) deletePod(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err == nil {
		event := Event{
			resource: kauloud.ResourceAbbrPod,
			verb: kauloud.WatcherEventVerbDelete,
			key: key,
		}
		o.logger.Debugf("deletion key : %s\n",key)
		o.Queue.Add(event)
	}
}

func (o *StreamingResourceDescriber) updatePod (old interface{}, new interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(new)
	if err == nil {
		event := Event{
			resource: kauloud.ResourceAbbrPod,
			verb: kauloud.WatcherEventVerbUpdate,
			key: key,
		}
		o.Queue.Add(event)
	}
}

func (o *StreamingResourceDescriber) addNode(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err == nil {
		event := Event{
			resource: kauloud.ResourceAbbrNode,
			verb: kauloud.WatcherEventVerbAdd,
			key: key,
		}
		o.Queue.Add(event)
	}
}

func (o *StreamingResourceDescriber) deleteNode(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err == nil {
		event := Event{
			resource: kauloud.ResourceAbbrNode,
			verb: kauloud.WatcherEventVerbDelete,
			key: key,
		}
		o.Queue.Add(event)
	}
}

func (o *StreamingResourceDescriber) updateNode (old interface{}, new interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(new)
	if err == nil {
		event := Event{
			resource: kauloud.ResourceAbbrNode,
			verb: kauloud.WatcherEventVerbUpdate,
			key: key,
		}
		o.Queue.Add(event)
	}
}

func (o *StreamingResourceDescriber) runTestLister(stopCh <-chan struct{}) {
	for {
		time.Sleep(10 * time.Second)
		if o.podList == nil {
			continue
		}
		o.logger.Debugf("---------- Requests ----------\n")
		for key, value := range o.requests.perNode {
			o.logger.Debugf("-- Node : %s --\n", key)
			o.logger.Debugf("Cpus := %s \n", value.Cpu().String())
			o.logger.Debugf("Memory := %s \n", value.Memory().String())
		}

		o.logger.Debugf("\n---------- Limits ----------\n")
		for key, value := range o.limits.perNode {
			o.logger.Debugf("-- Node : %s --\n", key)
			o.logger.Debugf("Cpus := %s \n", value.Cpu().String())
			o.logger.Debugf("Memory := %s \n", value.Memory().String())
		}
		o.logger.Debugf("\n ==> Summary \n")
	}
	<- stopCh
}

type ResourceMap struct {
	perNode map[string]corev1.ResourceList
	allNode corev1.ResourceList
}

func NewResourceMap () *ResourceMap {
	return &ResourceMap{
		perNode: make(map[string]corev1.ResourceList),
		allNode: make(corev1.ResourceList),
	}
}

func (rm *ResourceMap) RefNodeResourceList(nodeName string) (corev1.ResourceList, error) {
	if resources, ok := rm.perNode[nodeName]; !ok {
		return nil, errors.New("the resource list has no data with this node")
	} else {
		// if we need to return new resource object instead of origin map pointer,
		// return resources.DeepCopy()
		return resources, nil
	}
}

func (rm *ResourceMap) add(nodeName string, resources corev1.ResourceList) {
	if _, exists := rm.perNode[nodeName]; !exists {
		rm.perNode[nodeName] = make(corev1.ResourceList)
	}

	for resourceName, resourceQuantity := range resources {
		if value, ok := rm.perNode[nodeName][resourceName]; !ok {
			rm.perNode[nodeName][resourceName] = resourceQuantity.DeepCopy()
		} else {
			value.Add(resourceQuantity)
			rm.perNode[nodeName][resourceName] = value
		}

		if value, ok := rm.allNode[resourceName]; !ok {
			rm.allNode[resourceName] = resourceQuantity.DeepCopy()
		} else {
			value.Add(resourceQuantity)
			rm.allNode[resourceName] = value
		}
	}
}

func (rm *ResourceMap) sub(nodeName string, resources corev1.ResourceList) {
	if _, exists := rm.perNode[nodeName]; !exists {
		return
	}

	for resourceName, resourceQuantity := range resources {
		if value, ok := rm.perNode[nodeName][resourceName]; !ok {
			continue
		} else {
			value.Sub(resourceQuantity)
			rm.perNode[nodeName][resourceName] = value
		}

		if value, ok := rm.allNode[resourceName]; !ok {
			continue
		} else {
			value.Sub(resourceQuantity)
			rm.allNode[resourceName] = value
		}
	}
}

func (rm *ResourceMap) deleteNode(nodeName string) {
	resources, ok := rm.perNode[nodeName]
	if !ok {
		return
	}
	rm.sub(nodeName, resources)
	delete(rm.perNode, nodeName)
}
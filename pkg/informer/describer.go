package informer

import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/xjayleex/kauloud/pkg/api/kauloud"
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
	Availability(template interface{}) bool
	Nodes() map[string]*corev1.Node
	Requested() *ResourceMap
	Limits() *ResourceMap
	Allocatable()  *ResourceMap
	Run(int, chan struct{})
}

type PollingResourceDescriber struct {
	clientset *kubernetes.Clientset
	logger *logrus.Logger

	requests    *ResourceMap
	limits      *ResourceMap
	allocatable *ResourceMap

	nodeList map[string]*corev1.Node
	podList map[string]*corev1.Pod

	podInformer cache.SharedInformer
	nodeInformer cache.SharedInformer

	Queue workqueue.RateLimitingInterface
}

func NewPollingResourceDescriber (clientset *kubernetes.Clientset, logger *logrus.Logger) *PollingResourceDescriber {
	new := &PollingResourceDescriber{
		clientset:   clientset,
		logger:      logger,
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

func (o *PollingResourceDescriber) Run(threadiness int, stopCh chan struct{}) {
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
	<-stopCh
	o.logger.Infoln("Stopping describer.")
}

func (o *PollingResourceDescriber) runWorker() {
	for o.consume() {}
}

func (o *PollingResourceDescriber) consume() bool {
	key, quit := o.Queue.Get()
	if quit {
		return false
	}
	defer o.Queue.Done(key)

	err := o.sync(key.(EventKey))
	o.handleError(err, key)
	return true
}

func (o *PollingResourceDescriber) sync(eventKey EventKey) error {
	var object interface{}
	var exists bool
	var err error

	switch eventKey.resource {
	case kauloud.ResourceAbbrPod:
		object, exists, err = o.podInformer.GetStore().GetByKey(eventKey.key.(string))
	case kauloud.ResourceAbbrNode:
		object, exists, err = o.nodeInformer.GetStore().GetByKey(eventKey.key.(string))
	}

	if err != nil {
		o.logger.Infof("error fetching object from index for the specified key. %s %v", eventKey.key.(string), err)
		return err
	}

	if !exists {
		err := o.deleteHandler(object, &eventKey)
		return err
	}

	if eventKey.verb == kauloud.WatcherEventVerbAdd {
		err = o.addHandler(object, &eventKey)
	} else { // "update"
		err = o.updateHandler(object, &eventKey)
	}

	return err
}

func (o *PollingResourceDescriber) addHandler(object interface{}, eventKey *EventKey) (err error) {
	o.logger.Debugf("Add received for %s", eventKey.key.(string))
	switch eventKey.resource {
	case kauloud.ResourceAbbrPod:
		err = o.handlePodAddition(object)
	}
	return err
}

func (o *PollingResourceDescriber) deleteHandler(object interface{}, eventKey *EventKey) (err error) {
	o.logger.Debugf("Delete received for %s", eventKey.key.(string))
	switch eventKey.resource {
	case kauloud.ResourceAbbrPod:
		err = o.handlePodDeletion(object)
	}
	return err
}

func (o *PollingResourceDescriber) updateHandler(object interface{}, eventKey *EventKey) (err error) {
	o.logger.Debugf("Update received for %s", eventKey.key.(string))
	switch eventKey.resource {
	case kauloud.ResourceAbbrPod:
		err = o.handlePodUpdate(object)
	case kauloud.ResourceAbbrNode:
		o.handleNodeUpdate(object)
	}
	return err
}

func (o *PollingResourceDescriber) handlePodAddition (object interface{}) error {
	pod, ok := object.(*corev1.Pod)
	if !ok {
		return errors.New("error on type assertion for pod")
	}

	if pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodSucceeded  {
		return nil
	}

	o.podList[pod.Name] = pod
	requests, limits := PodRequestsAndLimits(pod)
	o.requests.add(pod.Spec.NodeName, requests)
	o.limits.add(pod.Spec.NodeName, limits)
	return nil
}

func (o *PollingResourceDescriber) handlePodDeletion (object interface{}) error {
	pod, ok := object.(*corev1.Pod)
	if !ok {
		return errors.New("error on type assertion for pod")
	}

	if _, exists := o.podList[pod.Name]; !exists {
		return nil
	}

	requests, limits := PodRequestsAndLimits(pod)
	o.requests.sub(pod.Spec.NodeName, requests)
	o.limits.sub(pod.Spec.NodeName, limits)
	delete(o.podList, pod.Name)

	return nil
}

func (o *PollingResourceDescriber) handlePodUpdate (object interface{}) error {
	pod, ok := object.(*corev1.Pod)
	if !ok {
		return errors.New("error on type assertion for pod")
	}

	if pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodSucceeded {
		if _, exists := o.podList[pod.Name]; exists {
			requests, limits := PodRequestsAndLimits(pod)
			o.requests.sub(pod.Spec.NodeName, requests)
			o.limits.sub(pod.Spec.NodeName, limits)
			delete(o.podList, pod.Name)
		}
		return nil
	}

	o.podList[pod.Name] = pod
	return nil
}

func (o *PollingResourceDescriber) handleNodeAddition (object interface{}) {
}

func (o *PollingResourceDescriber) handleNodeDeletion (object interface{}) {
}

func (o *PollingResourceDescriber) handleNodeUpdate (object interface{}) {
}

func (o *PollingResourceDescriber) handleError (err error, key interface{}) {
	if err == nil {
		o.Queue.Forget(key)
		return
	}

	if o.Queue.NumRequeues(key) < kauloud.ControllerMaxRequeue {
		o.logger.Infof("error during sync %s %v", key.(string), err)
		o.Queue.AddRateLimited(key)
		return
	}

	o.Queue.Forget(key)
	runtime.HandleError(err)
	o.logger.Infof("drop pod out of queue after many retries, %s %v",key.(string), err)
}

func (o *PollingResourceDescriber) addPod(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err == nil {
		eventKey := EventKey{
			resource: kauloud.ResourceAbbrPod,
			verb: kauloud.WatcherEventVerbAdd,
			key: key,
		}
		o.Queue.Add(eventKey)
	}
}

func (o *PollingResourceDescriber) deletePod(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err == nil {
		eventKey := EventKey{
			resource: kauloud.ResourceAbbrPod,
			verb: kauloud.WatcherEventVerbDelete,
			key: key,
		}
		o.Queue.Add(eventKey)
	}
}

func (o *PollingResourceDescriber) updatePod (old interface{}, new interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(new)
	if err == nil {
		eventKey := EventKey{
			resource: kauloud.ResourceAbbrPod,
			verb: kauloud.WatcherEventVerbUpdate,
			key: key,
		}
		o.Queue.Add(eventKey)
	}
}

func (o *PollingResourceDescriber) addNode(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err == nil {
		eventKey := EventKey{
			resource: kauloud.ResourceAbbrNode,
			verb: kauloud.WatcherEventVerbAdd,
			key: key,
		}
		o.Queue.Add(eventKey)
	}
}

func (o *PollingResourceDescriber) deleteNode(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err == nil {
		eventKey := EventKey{
			resource: kauloud.ResourceAbbrNode,
			verb: kauloud.WatcherEventVerbDelete,
			key: key,
		}
		o.Queue.Add(eventKey)
	}
}

func (o *PollingResourceDescriber) updateNode (old interface{}, new interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(new)
	if err == nil {
		eventKey := EventKey{
			resource: kauloud.ResourceAbbrNode,
			verb: kauloud.WatcherEventVerbUpdate,
			key: key,
		}
		o.Queue.Add(eventKey)
	}
}


func (o *PollingResourceDescriber) runTestLister(stopCh <-chan struct{}) {
	for {
		time.Sleep(30 * time.Second)
		if o.podList == nil {
			continue
		}
		o.logger.Printf("---------- Requests ----------\n")
		for key, value := range o.requests.perNode {
			o.logger.Printf("-- Node : %s --\n", key)
			o.logger.Printf("Cpus := %s \n", value.Cpu().String())
			o.logger.Printf("Memory := %s \n", value.Memory().String())
		}

		o.logger.Printf("---------- Limits ----------\n")
		for key, value := range o.requests.perNode {
			o.logger.Printf("-- Node : %s --\n", key)
			o.logger.Printf("Cpus := %s \n", value.Cpu().String())
			o.logger.Printf("Memory := %s \n", value.Memory().String())
		}
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

func (rm *ResourceMap) addNode(node *corev1.Node) {
	rm.perNode[node.Name] = make(corev1.ResourceList)

}

func (rm *ResourceMap) deleteNode (node *corev1.Node) {
	// Todo something
}

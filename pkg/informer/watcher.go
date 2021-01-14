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
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	virtv1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
	"time"
)

type Watcher struct {
	podInformer cache.SharedInformer
	vmInformer cache.SharedInformer
	dvInformer cache.SharedInformer
	svcInformer cache.SharedInformer

	logger *logrus.Logger

	Queue workqueue.RateLimitingInterface

	vmList         map[string]*virtv1.VirtualMachine
	podList        map[string]*corev1.Pod
	dataVolumeList map[string]*cdiv1.DataVolume

	vmListByUser map[string]map[string]*virtv1.VirtualMachine
}

func NewWatcher(client kubecli.KubevirtClient, logger *logrus.Logger) *Watcher {
	new := &Watcher{
		logger:         logger,
		Queue:          workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		vmList:         make(map[string]*virtv1.VirtualMachine),
		dataVolumeList: make(map[string]*cdiv1.DataVolume),

		vmListByUser: make(map[string]map[string]*virtv1.VirtualMachine),
	}

	podLW := cache.NewListWatchFromClient(client.CoreV1().RESTClient(),"pods", kauloud.TargetNamespace, fields.Everything())
	new.podInformer = cache.NewSharedInformer(podLW, &corev1.Pod{}, 0)
	new.podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    new.addPod,
		UpdateFunc: new.updatePod,
		DeleteFunc: new.deletePod,
	})

	vmLW := cache.NewListWatchFromClient(client.RestClient(), "virtualmachines", kauloud.TargetNamespace, fields.Everything())
	new.vmInformer = cache.NewSharedInformer(vmLW, &virtv1.VirtualMachine{}, 0)
	new.vmInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    new.addVm,
		UpdateFunc: new.updateVm,
		DeleteFunc: new.deleteVm,
	})

	dvLW := cache.NewListWatchFromClient(client.CdiClient().CdiV1alpha1().RESTClient(), "datavolumes", kauloud.TargetNamespace, fields.Everything())
	new.dvInformer = cache.NewSharedInformer(dvLW, &cdiv1.DataVolume{}, 0)
	new.dvInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: new.addDv,
		UpdateFunc: new.updateDv,
		DeleteFunc: new.deleteDv,
	})

	svcLW := cache.NewListWatchFromClient(client.CoreV1().RESTClient(), "services", kauloud.TargetNamespace, fields.Everything())
	new.svcInformer = cache.NewSharedInformer(svcLW, &corev1.Service{}, 0)
	new.svcInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: new.addSvc,
		UpdateFunc: new.updateSvc,
		DeleteFunc: new.deleteSvc,
	})

	return new
}

func (w *Watcher) Logger() *logrus.Logger {
	return w.logger
}

func (w *Watcher) VmList() map[string]*virtv1.VirtualMachine {
	return w.vmList
}

func (w *Watcher) PodList() map[string]*corev1.Pod {
	return w.podList
}

func (w *Watcher) DataVolumeList() map[string]*cdiv1.DataVolume {
	return w.dataVolumeList
}

func (w *Watcher) VmListByUser() map[string]map[string]*virtv1.VirtualMachine {
	return w.vmListByUser
}

func (w *Watcher) Run (threadiness int, stopCh chan struct{}) {
	defer runtime.HandleCrash()
	defer w.Queue.ShutDown()
	w.logger.Infoln("Starting controller.")
	go w.podInformer.Run(stopCh)
	go w.vmInformer.Run(stopCh)
	go w.dvInformer.Run(stopCh)
	go w.runTestLister2(stopCh)
	if !cache.WaitForCacheSync(stopCh, w.podInformer.HasSynced, w.vmInformer.HasSynced, w.dvInformer.HasSynced) {
		runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}

	for i := 0; i < threadiness; i++ {
		w.logger.Infoln("Run watch dogs for watcher.")
		go wait.Until(w.runWorker, time.Second, stopCh)
	}
	<-stopCh
	w.logger.Infoln("Stopping watcher.")
}

func (w *Watcher) runWorker() {
	for w.consume() {}
}

func (w *Watcher) consume() bool {
	key, quit := w.Queue.Get()
	if quit {
		return false
	}
	defer w.Queue.Done(key)

	err := w.sync(key.(EventKey))
	w.handleError(err, key)
	return true
}

func (w *Watcher) sync(eventKey EventKey) error {
	var object interface{}
	var exists bool
	var err	error

	switch eventKey.resource {
	case kauloud.ResourceAbbrVirtualMachine:
		object, exists, err = w.vmInformer.GetStore().GetByKey(eventKey.key.(string))
	case kauloud.ResourceAbbrPod:
		object, exists, err = w.podInformer.GetStore().GetByKey(eventKey.key.(string))
	case kauloud.ResourceAbbrDataVolume:
		object, exists, err = w.dvInformer.GetStore().GetByKey(eventKey.key.(string))
	}

	if err != nil {
		w.logger.Infof("error fetching object from index for the specified key. %s %v", eventKey.key.(string), err)
		return err
	}

	if !exists {
		err := w.deleteHandle(object, &eventKey)
		return err
	}

	if eventKey.verb == kauloud.WatcherEventVerbAdd {
		err = w.addHandle(object, &eventKey)
	} else { // "update"
		err = w.updateHandle(object, &eventKey)
	}

	return err
}

func (w *Watcher) handleError (err error, key interface{}) {
	if err == nil {
		w.Queue.Forget(key)
		return
	}

	if w.Queue.NumRequeues(key) < kauloud.ControllerMaxRequeue {
		w.logger.Infof("error during sync %s %v", key.(string), err)
		w.Queue.AddRateLimited(key)
		return
	}

	w.Queue.Forget(key)
	runtime.HandleError(err)
	w.logger.Infof("drop pod out of queue after many retries, %s %v",key.(string), err)
}

func (w *Watcher) runTestLister (stopCh <-chan struct{}) {
	for {
		time.Sleep(10 * time.Second)
		if w.vmList == nil {
			continue
		}

		for key := range w.vmList {
			vmstat, err := w.refVmStatus(key)
			if err != nil {
				w.logger.Infof("error on listing the vm name is : %s\n",key)
				continue
			}
			w.logger.Infof("Listing : %s\n", vmstat.String())
		}
	}
	<- stopCh
}

func (w *Watcher) runTestLister2 (stopCh <-chan struct{}) {
	for {
		time.Sleep(10 * time.Second)
		if w.vmListByUser == nil {
			w.logger.Infof("Error on listing ...\n")
			continue
		}

		for key, value := range w.vmListByUser {
			w.logger.Infof("Listing for %s ...\n", key)
			for insideKey := range value {
				vmstat, err := w.refVmStatus(insideKey)
				if err != nil {
					w.logger.Infof("... error on listing the vm name is : %s\n",key)
					continue
				}
				w.logger.Infof("... %s\n", vmstat.String())
			}
		}
	}
	<- stopCh
}

func (w *Watcher) addPod(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err == nil {
		eventKey := EventKey{
			resource: kauloud.ResourceAbbrPod,
			verb: kauloud.WatcherEventVerbAdd,
			key: key,
		}
		w.Queue.Add(eventKey)
	}
}

func (w *Watcher) deletePod(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err == nil {
		eventKey := EventKey{
			resource: kauloud.ResourceAbbrPod,
			verb: kauloud.WatcherEventVerbDelete,
			key: key,
		}
		w.Queue.Add(eventKey)
	}
}

func (w *Watcher) updatePod (old interface{}, new interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(new)
	if err == nil {
		eventKey := EventKey{
			resource: kauloud.ResourceAbbrPod,
			verb: kauloud.WatcherEventVerbUpdate,
			key: key,
		}
		w.Queue.Add(eventKey)
	}
}

func (w *Watcher) addVm(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err == nil {
		eventKey := EventKey{
			resource: kauloud.ResourceAbbrVirtualMachine,
			verb: kauloud.WatcherEventVerbAdd,
			key: key,
		}
		w.Queue.Add(eventKey)
	}
}

func (w *Watcher) deleteVm(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err == nil {
		eventKey := EventKey{
			resource: kauloud.ResourceAbbrVirtualMachine,
			verb: kauloud.WatcherEventVerbDelete,
			key: key,
		}
		w.Queue.Add(eventKey)
	}
}

func (w *Watcher) updateVm (old interface{}, new interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(new)
	if err == nil {
		eventKey := EventKey{
			resource: kauloud.ResourceAbbrVirtualMachine,
			verb: kauloud.WatcherEventVerbUpdate,
			key: key,
		}
		w.Queue.Add(eventKey)
	}
}

func (w *Watcher) addDv(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err == nil {
		eventKey := EventKey{
			resource: kauloud.ResourceAbbrDataVolume,
			verb: kauloud.WatcherEventVerbAdd,
			key: key,
		}
		w.Queue.Add(eventKey)
	}
}

func (w *Watcher) deleteDv(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err == nil {
		eventKey := EventKey{
			resource: kauloud.ResourceAbbrDataVolume,
			verb: kauloud.WatcherEventVerbDelete,
			key: key,
		}
		w.Queue.Add(eventKey)
	}
}

func (w *Watcher) updateDv(old interface{}, new interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(new)
	if err == nil {
		eventKey := EventKey{
			resource: kauloud.ResourceAbbrDataVolume,
			verb: kauloud.WatcherEventVerbUpdate,
			key: key,
		}
		w.Queue.Add(eventKey)
	}
}

func (w *Watcher) addSvc(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err == nil {
		eventKey := EventKey{
			resource: kauloud.ResourceAbbrService,
			verb: kauloud.WatcherEventVerbAdd,
			key: key,
		}
		w.Queue.Add(eventKey)
	}
}

func (w *Watcher) deleteSvc(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err == nil {
		eventKey := EventKey{
			resource: kauloud.ResourceAbbrService,
			verb: kauloud.WatcherEventVerbDelete,
			key: key,
		}
		w.Queue.Add(eventKey)
	}
}

func (w *Watcher) updateSvc(old interface{}, new interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(new)
	if err == nil {
		eventKey := EventKey{
			resource: kauloud.ResourceAbbrService,
			verb: kauloud.WatcherEventVerbUpdate,
			key: key,
		}
		w.Queue.Add(eventKey)
	}
}

func (w *Watcher) refVmStatus (vmName string) (*VirtualMachineStatus, error){
	vm, ok := w.vmList[vmName]
	if !ok {
		return nil, errors.New("vm does not exists")
	}
	vmStatus := &VirtualMachineStatus{}

	if len(vm.Spec.DataVolumeTemplates) == 0 {
		return nil, errors.New("dv does not exists")
	}
	dv, ok := w.dataVolumeList[vm.Spec.DataVolumeTemplates[0].Name]
	if !ok {
		return nil, errors.New("dv does not exists")
	}

	if dv.Status.Phase == kauloud.DataVolumeSucceed {
		vmStatus.isReady = true
	} else {
		vmStatus.isReady = false
	}
	vmStatus.vm = vm
	vmStatus.progress = string(dv.Status.Progress)
	vmStatus.isRunning = vm.Status.Created
	return vmStatus, nil
}

func (w *Watcher) addHandle (object interface{}, eventKey *EventKey) (err error) {
	w.logger.Debugf("Add received for %s", eventKey.key.(string))
	switch eventKey.resource {
	case kauloud.ResourceAbbrDataVolume:
		dv, ok := object.(*cdiv1.DataVolume)
		if !ok {
			return nil
		}
		w.dataVolumeList[dv.Name] = dv
	case kauloud.ResourceAbbrVirtualMachine:
		vm, ok := object.(*virtv1.VirtualMachine)
		if !ok {
			return nil
		}
		w.vmList[vm.Name] = vm
		///////////////////////////////////////
		userId, ok := vm.Labels[kauloud.LabelKeyKauloudUserId]
		if ok {
			if w.vmListByUser[userId] == nil {
				w.vmListByUser[userId] = map[string]*virtv1.VirtualMachine{}
			}
			w.vmListByUser[userId][vm.Name] = vm
		}
		///////////////////////////////////////
	case kauloud.ResourceAbbrService:
		// TODO : svc watch handle

	}
	return err
}

func (w *Watcher) deleteHandle (object interface{}, eventKey *EventKey) (err error)  {
	w.logger.Debugf("obj has deleted %s", eventKey.key.(string))

	switch eventKey.resource {
	case kauloud.ResourceAbbrDataVolume:
		dv, ok := object.(*cdiv1.DataVolume)
		if !ok {
			return nil
		}
		delete(w.dataVolumeList, dv.Name)
	case kauloud.ResourceAbbrVirtualMachine:
		vm, ok := object.(*virtv1.VirtualMachine)
		if !ok {
			return nil
		}
		delete(w.vmList, vm.Name)
		///////////////////////////////////////
		userId, ok := vm.Labels[kauloud.LabelKeyKauloudUserId]
		if ok {
			vmForUser := w.vmListByUser[userId]
			delete(vmForUser, vm.Name)
		}
		///////////////////////////////////////
	}

	return err
}

func (w *Watcher) updateHandle (object interface{}, eventKey *EventKey) (err error) {
	w.logger.Infof("Update received for %s", eventKey.key.(string))
	switch eventKey.resource {
	case kauloud.ResourceAbbrDataVolume:
		dv, ok := object.(*cdiv1.DataVolume)
		if !ok {
			return nil
		}
		w.dataVolumeList[dv.Name] = dv
	case kauloud.ResourceAbbrVirtualMachine:
		vm, ok := object.(*virtv1.VirtualMachine)
		if !ok {
			return nil
		}
		w.vmList[vm.Name] = vm
		///////////////////////////////////////
		userId, ok := vm.Labels[kauloud.LabelKeyKauloudUserId]
		if ok {
			if w.vmListByUser[userId] == nil {
				w.vmListByUser[userId] = map[string]*virtv1.VirtualMachine{}
			}
			w.vmListByUser[userId][vm.Name] = vm
		}
		///////////////////////////////////////
	}
	return err
}


type EventKey struct {
	resource string
	verb string
	key interface{}
}

type VirtualMachineStatus struct {
	isReady   bool
	progress  string
	isRunning bool
	vm *virtv1.VirtualMachine
}

func (vms *VirtualMachineStatus) IsReady() bool{
	return vms.isReady
}

func (vms *VirtualMachineStatus) Progress() string{
	return vms.progress
}

func (vms *VirtualMachineStatus) IsRunning() bool{
	return vms.isRunning
}

func (vms *VirtualMachineStatus) VirtualMachine() *virtv1.VirtualMachine {
	return vms.vm
}

func (vms *VirtualMachineStatus) String() string{
	return fmt.Sprintf("%s :: isReady (%v), IsProgress(%s), isRunning(%v)", vms.VirtualMachine().Name, vms.IsReady(), vms.Progress(), vms.IsRunning())
}

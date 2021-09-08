package informer

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/xjayleex/kauloud/pkg/api/kauloud"
	"github.com/xjayleex/kauloud/pkg/utils"
	pb "github.com/xjayleex/kauloud/proto"
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
	logger          *logrus.Logger
	Queue           workqueue.RateLimitingInterface
	running         bool
	workloadMetaMap map[string]*kauloud.WorkloadMeta
	vmInfoList      VirtualMachineInfoList

	podInformer cache.SharedInformer
	vmInformer cache.SharedInformer
	dvInformer cache.SharedInformer
	svcInformer cache.SharedInformer
}

func NewWatcher(client kubecli.KubevirtClient, logger *logrus.Logger) *Watcher {
	new := &Watcher{
		logger:          logger,
		Queue:           workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		running:         false,
		workloadMetaMap: make(map[string]*kauloud.WorkloadMeta),
		vmInfoList:      NewVirtualMachineInfoList(),
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

func (w *Watcher) VmInfoList() VirtualMachineInfoList {
	return w.vmInfoList
}

func (w *Watcher) Run (threadiness int, stopCh chan struct{}) {
	defer runtime.HandleCrash()
	defer w.Queue.ShutDown()
	w.logger.Infoln("Starting Watcher.")
	go w.podInformer.Run(stopCh)
	go w.vmInformer.Run(stopCh)
	go w.dvInformer.Run(stopCh)
	go w.svcInformer.Run(stopCh)

	if !cache.WaitForCacheSync(stopCh, w.podInformer.HasSynced, w.vmInformer.HasSynced, w.dvInformer.HasSynced, w.svcInformer.HasSynced) {
		runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}

	for i := 0; i < threadiness; i++ {
		w.logger.Infoln("Run watch dogs for watcher.")
		go wait.Until(w.runWorker, time.Second, stopCh)
	}
	w.running = true
	defer func() {w.running = false}()
	<-stopCh
	w.logger.Infoln("Stopping watcher.")
}

func (w *Watcher) GetUserVirtualMachineInfoList (userid kauloud.UserID) (map[kauloud.UUID]*VirtualMachineInfo, error) {
	return w.vmInfoList.GetUserVirtualMachineInfoList(userid)
}

func (w *Watcher) GetVirtualMachineInfo (userid kauloud.UserID, uuid kauloud.UUID) (*VirtualMachineInfo, error) {
	return w.vmInfoList.GetVirtualMachineInfo(userid, uuid)
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

	err := w.sync(key.(Event))
	w.handleError(err, key)
	return true
}

func (w *Watcher) sync(event Event) error {
	var object interface{}
	var exists bool
	var err	error

	switch event.resource {
	case kauloud.ResourceAbbrVirtualMachine:
		object, exists, err = w.vmInformer.GetStore().GetByKey(event.GetKey())
	case kauloud.ResourceAbbrPod:
		object, exists, err = w.podInformer.GetStore().GetByKey(event.GetKey())
	case kauloud.ResourceAbbrDataVolume:
		object, exists, err = w.dvInformer.GetStore().GetByKey(event.GetKey())
	case kauloud.ResourceAbbrService:
		object, exists, err = w.svcInformer.GetStore().GetByKey(event.GetKey())
	}

	if err != nil {
		w.logger.Infof("error fetching object from index for the specified key. %s %v", event.GetKey(), err)
		return err
	}

	if !exists {
		err := w.deleteHandler(&event)
		return err
	}

	if event.verb == kauloud.WatcherEventVerbAdd {
		err = w.addHandler(object, &event)
	} else { // "update"
		err = w.updateHandler(object, &event)
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

func (w *Watcher) addPod(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err == nil {
		event := Event{
			resource: kauloud.ResourceAbbrPod,
			verb: kauloud.WatcherEventVerbAdd,
			key: key,
		}
		w.Queue.Add(event)
	}
}

func (w *Watcher) deletePod(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err == nil {
		event := Event{
			resource: kauloud.ResourceAbbrPod,
			verb: kauloud.WatcherEventVerbDelete,
			key: key,
		}
		w.Queue.Add(event)
	}
}

func (w *Watcher) updatePod (old interface{}, new interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(new)
	if err == nil {
		event := Event{
			resource: kauloud.ResourceAbbrPod,
			verb: kauloud.WatcherEventVerbUpdate,
			key: key,
		}
		w.Queue.Add(event)
	}
}

func (w *Watcher) addVm(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err == nil {
		event := Event{
			resource: kauloud.ResourceAbbrVirtualMachine,
			verb: kauloud.WatcherEventVerbAdd,
			key: key,
		}
		w.Queue.Add(event)
	}
}

func (w *Watcher) deleteVm(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err == nil {
		event := Event{
			resource: kauloud.ResourceAbbrVirtualMachine,
			verb: kauloud.WatcherEventVerbDelete,
			key: key,
		}
		w.Queue.Add(event)
	}
}

func (w *Watcher) updateVm (old interface{}, new interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(new)
	if err == nil {
		event := Event{
			resource: kauloud.ResourceAbbrVirtualMachine,
			verb: kauloud.WatcherEventVerbUpdate,
			key: key,
		}
		w.Queue.Add(event)
	}
}

func (w *Watcher) addDv(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err == nil {
		event := Event{
			resource: kauloud.ResourceAbbrDataVolume,
			verb: kauloud.WatcherEventVerbAdd,
			key: key,
		}
		w.Queue.Add(event)
	}
}

func (w *Watcher) deleteDv(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err == nil {
		event := Event{
			resource: kauloud.ResourceAbbrDataVolume,
			verb: kauloud.WatcherEventVerbDelete,
			key: key,
		}
		w.Queue.Add(event)
	}
}

func (w *Watcher) updateDv(old interface{}, new interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(new)
	if err == nil {
		event := Event{
			resource: kauloud.ResourceAbbrDataVolume,
			verb: kauloud.WatcherEventVerbUpdate,
			key: key,
		}
		w.Queue.Add(event)
	}
}

func (w *Watcher) addSvc(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err == nil {
		event := Event{
			resource: kauloud.ResourceAbbrService,
			verb: kauloud.WatcherEventVerbAdd,
			key: key,
		}
		w.Queue.Add(event)
	}
}

func (w *Watcher) deleteSvc(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err == nil {
		event := Event{
			resource: kauloud.ResourceAbbrService,
			verb: kauloud.WatcherEventVerbDelete,
			key: key,
		}
		w.Queue.Add(event)
	}
}

func (w *Watcher) updateSvc(old interface{}, new interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(new)
	if err == nil {
		event := Event{
			resource: kauloud.ResourceAbbrService,
			verb: kauloud.WatcherEventVerbUpdate,
			key: key,
		}
		w.Queue.Add(event)
	}
}

func (w *Watcher) handleVirtualMachineUpdate(object interface{}, event *Event) error {
	vm, ok := object.(*virtv1.VirtualMachine)
	if !ok {
		return kauloud.ErrorTypeAssertionForVM
	}

	meta := utils.ExtractWorkloadMetaFromLabels(vm.Labels)

	// Because userid and uuid never change, maybe we need to do nothing when key exists.
	if _, ok := w.workloadMetaMap[event.GetKey()]; !ok {
		w.workloadMetaMap[event.GetKey()] = meta
	}

	target := w.vmInfoList.referVirtualMachineInfo(meta)
	target.Object.SetVirtualMachine(vm)
	target.updateStatOnVirtualMachineUpdate()
	return nil
}

func (w *Watcher) handleServiceUpdate(object interface{}) error {
	service, ok := object.(*corev1.Service)
	if !ok {
		return kauloud.ErrorTypeAssertionForService
	}

	serviceType := service.Spec.Type
	meta := utils.ExtractWorkloadMetaFromLabels(service.Labels)

	target := w.vmInfoList.referVirtualMachineInfo(meta)
	switch serviceType {
	case corev1.ServiceTypeClusterIP:
		target.Object.SetClusterIPService(service)
		target.updateStatOnClusterIPServiceUpdate()
	}
	return nil
}

// This function must handle both BASE IMAGE DataVolume and USER VM DataVolume.
// If all VirtualMachines are created by our system, we can be sure that at least
// USER VM DataVolume has label for `kauloud.LabelKeyKauloudDvType`, and the matched value is
// `kauloud.LabelValueKauloudDvTypeUserVmImage`.
// 1. BASE IMAGE DataVolume Added or Updated
// 2. USER VM DataVolume with appropriate labels Added or Updated
// 3. USER VM DataVolume with non-appropriate labels Added or Updated
// ** NOTE ** 'appropriate' means that DataVolume has label for 'LabelKeyKauloudDvType'
func (w *Watcher) handleDataVolumeUpdate(object interface{}) error {
	datavolume, ok := object.(*cdiv1.DataVolume)
	if !ok {
		return kauloud.ErrorTypeAssertionForDatavolume
	}
	// Before extracting metadata, we must identify that the datavolume is
	// base image, or user vm datavolume.
	if dvType, ok := datavolume.Labels[kauloud.LabelKeyKauloudDvType]; ok {
		switch dvType {
		case kauloud.LabelValueKauloudDvTypeBaseImage:
			// if we need to manage image metadata on 'Watcher',
			// we can write code more on this loop.
			return nil
		case kauloud.LabelValueKauloudDvTypeUserVmImage:
			meta := utils.ExtractWorkloadMetaFromLabels(datavolume.Labels)
			target := w.vmInfoList.referVirtualMachineInfo(meta)
			target.Object.SetDataVolume(datavolume)
			target.updateStatOnDataVolumeUpdate()
			return nil
		default:
			// If condition reaches here, it means the datavolume has non-appropriate
			// label value for the DataVolumeType. Then, we don't return, and process some handling
			// for this exception from below codes.
		}
	}
	// DataVolume has no label key for DataVolume Type reaches here.
	// Such DataVolume is one of the following.
	// 1. Uploaded with 'virtctl image-upload'
	// 2. Cloned with no DataVolumeType label from no-appropriate virtual machine yaml specification, or the other.

	return nil
}
// When Virtual machine deleted, we need to delete all metadata about the workload. (workloadMetaMap entry, VmStatusMap entry)
// 1. Get workloadMetaMap by event key and delete this associated entry from workloadMetaMap.
// 2. Delete associated entry from VmStatusMap with Metadata.
// 3. Delete all pointer to the workload objects.
func (w *Watcher) handleVirtualMachineDeletion(event *Event) error {
	meta, ok := w.workloadMetaMap[event.GetKey()]
	if !ok {
		return nil
	}
	delete(w.workloadMetaMap, event.GetKey())
	deleted, err := w.vmInfoList.deleteVirtualMachineInfo(meta)
	if err != nil {
		return nil
	}
	deleted.updateStatOnVirtualMachineDeletion()
	meta, deleted = nil, nil
	return nil
}

// Service deletion must affect only specific service with serviceType
func (w *Watcher) handleServiceDeletion(event *Event) error {
	meta, ok := w.workloadMetaMap[event.GetKey()]
	if !ok {
		return nil
	}
	if _, ok := w.vmInfoList[meta.Owner]; !ok {
		return nil
	}
	target, ok := w.vmInfoList[meta.Owner][meta.UUID]
	if !ok {
		return nil
	}
	serviceType, err := utils.GetServiceTypeFromEventKey(event.key.(string))
	if err != nil {
		return err
	}
	switch serviceType {
	case corev1.ServiceTypeClusterIP:
		target.updateStatOnClusterIPServiceDeletion()
	}
	return nil
}

func (w *Watcher) handleDataVolumeDeletion(event *Event) error {
	meta, ok := w.workloadMetaMap[event.GetKey()]
	if !ok {
		return nil
	}
	if _, ok := w.vmInfoList[meta.Owner]; !ok {
		return nil
	}
	target, ok := w.vmInfoList[meta.Owner][meta.UUID]
	if !ok {
		return nil
	}
	target.updateStatOnDataVolumeDeletion()
	return nil
}

func (w *Watcher) addHandler(object interface{}, event *Event) (err error) {
	w.logger.Debugf("Add received for %s", event.GetKey())
	switch event.resource {
	case kauloud.ResourceAbbrDataVolume:
		err = w.handleDataVolumeUpdate(object)
	case kauloud.ResourceAbbrVirtualMachine:
		err = w.handleVirtualMachineUpdate(object, event)
	case kauloud.ResourceAbbrService:
		err = w.handleServiceUpdate(object)
	}
	return err
}

func (w *Watcher) deleteHandler (event *Event) (err error)  {
	w.logger.Debugf("obj has deleted %s", event.GetKey())

	switch event.resource {
	case kauloud.ResourceAbbrDataVolume:
		err = w.handleDataVolumeDeletion(event)
	case kauloud.ResourceAbbrVirtualMachine:
		err = w.handleVirtualMachineDeletion(event)
	case kauloud.ResourceAbbrService:
		err = w.handleServiceDeletion(event)
	}

	return err
}

func (w *Watcher) updateHandler (object interface{}, event *Event) (err error) {
	w.logger.Infof("Update received for %s", event.GetKey())
	switch event.resource {
	case kauloud.ResourceAbbrDataVolume:
		err = w.handleDataVolumeUpdate(object)
	case kauloud.ResourceAbbrVirtualMachine:
		err = w.handleVirtualMachineUpdate(object, event)
	case kauloud.ResourceAbbrService:
		err = w.handleServiceUpdate(object)
	}
	return err
}

type Event struct {
	resource string
	verb string
	key interface{}
}

func (e *Event) GetKey() string {
	return e.key.(string)
}

type VirtualMachineInfoList map[kauloud.UserID]map[kauloud.UUID]*VirtualMachineInfo

func NewVirtualMachineInfoList() map[kauloud.UserID]map[kauloud.UUID]*VirtualMachineInfo {
	return make(map[kauloud.UserID]map[kauloud.UUID]*VirtualMachineInfo)
}

func (list VirtualMachineInfoList) GetUserVirtualMachineInfoList(userid kauloud.UserID) (map[kauloud.UUID]*VirtualMachineInfo, error) {
	out, ok := list[userid]
	if !ok {
		return nil, fmt.Errorf("no virtual machine for user : %s", userid)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no virtual machine for user : %s", userid)
	}
	return out, nil
}

func (list VirtualMachineInfoList) GetVirtualMachineInfo(userid kauloud.UserID, uuid kauloud.UUID) (*VirtualMachineInfo, error) {
	perUserList, ok := list[userid]
	if !ok {
		return nil, fmt.Errorf("no virtual machine for user : %s", userid)
	}
	info, ok := perUserList[uuid]
	if !ok {
		return nil, fmt.Errorf("no virtual machine for uuid : %s", uuid)
	}
	return info, nil
}

// Memory allocation safe. refer the virtual machine information with workload metadata.
// if workload metadata associated VirtualMachineInfo does not exist, create a new VirtualMachineInfo
// and return it.
func (list VirtualMachineInfoList) referVirtualMachineInfo(meta *kauloud.WorkloadMeta) *VirtualMachineInfo {
	if _, ok := list[meta.Owner]; !ok {
		list[meta.Owner] = make(map[kauloud.UUID]*VirtualMachineInfo)
	}
	if _, ok := list[meta.Owner][meta.UUID]; !ok {
		list[meta.Owner][meta.UUID] = NewVirtualMachineInfo(meta)
	}
	return list[meta.Owner][meta.UUID]
}

// Delete workload metadata associated VirtualMachineInfo, and return deleted target.
func (list VirtualMachineInfoList) deleteVirtualMachineInfo(meta *kauloud.WorkloadMeta) (*VirtualMachineInfo, error){
	if _, ok := list[meta.Owner]; !ok {
		return nil, fmt.Errorf("no matching VirtualMachineInfo")
	}
	deleted, ok := list[meta.Owner][meta.UUID];
	if !ok {
		return nil, fmt.Errorf("no matching VirtualMachineInfo")
	}
	delete(list[meta.Owner], meta.UUID)
	return deleted, nil
}

type VirtualMachineInfo struct {
	Object *VirtualMachineObject
	Status *pb.VirtualMachineStatus
}

func NewVirtualMachineInfo (meta *kauloud.WorkloadMeta) *VirtualMachineInfo {
	return &VirtualMachineInfo{
		Object: &VirtualMachineObject{
			vm:         nil,
			datavolume: nil,
			clusterIP:  nil,
		},
		Status:  &pb.VirtualMachineStatus{
			IsReady:   false,
			Progress:  kauloud.None,
			IsRunning: false,
			ClusterIp: kauloud.None,
			UUID:      meta.UUID.String(),
			Owner:     meta.Owner.String(),
		},
	}
}

func (info *VirtualMachineInfo) updateStatOnVirtualMachineUpdate() {
	if info.Object.VirtualMachine() == nil {
		return
	}
	info.Status.IsRunning = info.Object.VirtualMachine().Status.Created
}

func (info *VirtualMachineInfo) updateStatOnDataVolumeUpdate() {
	if info.Object.DataVolume() == nil {
		return
	}
	if info.Object.DataVolume().Status.Phase == cdiv1.Succeeded {
		info.Status.IsReady = true
	} else {
		info.Status.IsReady = false
	}
	info.Status.Progress = string(info.Object.DataVolume().Status.Progress)
}

func (info *VirtualMachineInfo) updateStatOnClusterIPServiceUpdate() {
	if info.Object.ClusterIPService() == nil {
		return
	}
	info.Status.ClusterIp = info.Object.ClusterIPService().Spec.ClusterIP
}

func (info *VirtualMachineInfo) updateStatOnVirtualMachineDeletion() {
	/* info.vm = nil
	info.datavolume = nil
	info.clusterIP = nil */
	info.Object = nil
	info.Status = nil
}

func (info *VirtualMachineInfo) updateStatOnDataVolumeDeletion() {
	if info.Status == nil {
		return
	}
	info.Status.IsReady = false
	info.Status.Progress = kauloud.DataVolumeDeleted
}

func (info *VirtualMachineInfo) updateStatOnClusterIPServiceDeletion() {
	if info.Status == nil {
		return
	}
	info.Status.ClusterIp = kauloud.None
}

type VirtualMachineObject struct {
	vm			*virtv1.VirtualMachine
	datavolume	*cdiv1.DataVolume
	clusterIP	*corev1.Service
}

func (o *VirtualMachineObject) VirtualMachine() *virtv1.VirtualMachine {
	return o.vm
}
func (o *VirtualMachineObject) DataVolume() *cdiv1.DataVolume {
	return o.datavolume
}
func (o *VirtualMachineObject) ClusterIPService() *corev1.Service {
	return o.clusterIP
}
func (o *VirtualMachineObject) SetVirtualMachine(vm *virtv1.VirtualMachine) {
	o.vm = vm
}
func (o *VirtualMachineObject) SetDataVolume(datavolume *cdiv1.DataVolume) {
	o.datavolume = datavolume
}
func (o *VirtualMachineObject) SetClusterIPService(clusterIP *corev1.Service) {
	o.clusterIP = clusterIP
}
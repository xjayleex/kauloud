package virt

import (
	"errors"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/xjayleex/kauloud/pkg/api/kauloud"
	"github.com/xjayleex/kauloud/pkg/utils"
	pb "github.com/xjayleex/kauloud/proto"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	virtv1 "kubevirt.io/client-go/api/v1"
	cdiclient "kubevirt.io/client-go/generated/containerized-data-importer/clientset/versioned"
	"kubevirt.io/client-go/kubecli"
	cdiv1alpha1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
)

type Manager struct {
	config          	*utils.KauloudConfig
	logger          	*logrus.Logger
	client          	kubecli.KubevirtClient
	workloadManager 	*VMWorkloadManager
	imageManager    	*ImageManager
	instanceTypeLister	*InstanceTypeLister
}

func NewManager(config *utils.KauloudConfig, logger *logrus.Logger, virtClient kubecli.KubevirtClient) (*Manager, error){
	// TODO :: Client Config from outside args
	new := &Manager{
		config: config,
		logger: logger,
		client: virtClient,
	}

	imageManager, err := NewImageManager(logger, new.Client().CdiClient(), config)
	workloadManager := NewVMWorkloadManager(new.config)

	new.imageManager = imageManager
	new.workloadManager = workloadManager
	new.instanceTypeLister = NewInstanceTypeLister(DefaultInstanceTypes()...)

	return new, err
}

func (m *Manager) WorkloadManager() *VMWorkloadManager {
	return m.workloadManager
}

func (m *Manager) ImageManager() *ImageManager {
	return m.imageManager
}

func (m *Manager) Logger() *logrus.Logger {
	return m.logger
}

func (m *Manager) Client() kubecli.KubevirtClient {
	return m.client
}

func (m *Manager) CreateVirtualMachineWorkload(req VmCreationTemplate) error {
	imageMeta, ok := m.imageManager.ReferBaseImageMeta(req.ImageCode)
	if !ok {
		return errors.New("no matching images")
	}

	template := &ParsedVmCreationTemplate{
		VmCreationTemplate: &req,
		BaseImageMeta:      imageMeta,
		UUID:               uuid.New(),
	}
	workload := m.workloadManager.NewVmWorkload(template)
	clusterIP, err := m.Client().CoreV1().Services(m.config.VirtManagerConfig.Namespace).Create(workload.ClusterIP)
	if err != nil {
		return err
	}

	_, err = m.Client().VirtualMachine(m.config.VirtManagerConfig.Namespace).Create(workload.VM)
	if err != nil {
		err = m.Client().CoreV1().Services(m.config.VirtManagerConfig.Namespace).Delete(clusterIP.Name, &metav1.DeleteOptions{})
		return err
	}
	return err
}

func (m *Manager) DeleteVirtualMachineWorkload(meta *pb.WorkloadObjectNameMeta, options *metav1.DeleteOptions) error {
	// First, we try to delete Services. To do this, we should be able to get Service name by VirtualMachine information.
	err := m.Client().CoreV1().Services(m.config.VirtManagerConfig.Namespace).Delete(meta.ClusterIp, options)
	err = m.Client().VirtualMachine(m.config.VirtManagerConfig.Namespace).Delete(meta.VirtualMachine, options)
	return err
}

type ImageManager struct {
	cdiclient.Interface
	logger			*logrus.Logger
	config			*utils.KauloudConfig
	baseImageMetaList map[VirtualMachineImageCode]*BaseImageMeta
}

func NewImageManager (logger *logrus.Logger, cdiclient cdiclient.Interface, config *utils.KauloudConfig) (*ImageManager, error) {
	new := &ImageManager{
		logger: logger,
		Interface: cdiclient,
		config:    config,
	}

	err := new.loadBaseImageMetaList()

	return new, err
}

// Initialize BaseImageMetaList.
// Firstly, get DataVolume list from api server.
// And, check DataVolume's annotation with kauloud.LabelKeyKauloudImageCode for each item.
// If not exist, just skip.
// Then, check original BaseImageMetaList if the ImageCode is existing already.
// If not exist, store new BaseImageMeta gotten from the DataVolume's annotations.
func (m *ImageManager) loadBaseImageMetaList() error {
	m.baseImageMetaList = make(map[VirtualMachineImageCode]*BaseImageMeta)
	dvTypeLabel := map[string]string {
		kauloud.LabelKeyKauloudDvType: kauloud.LabelValueKauloudDvTypeBaseImage,
	}
	baseImages, err := m.listDataVolumeByLabel(dvTypeLabel)
	if err != nil {
		return err
	}
	for _, datavolume := range baseImages.Items {
		// Todo :: Must check it works. I changed ImageCode's store from Labels to Annotations.
		codeValue, ok := datavolume.Annotations[kauloud.LabelKeyKauloudImageCode]
		if !ok {
			continue
		}
		imageCode := VirtualMachineImageCode(codeValue)
		_, exists := m.ReferBaseImageMeta(imageCode)
		if exists {
			m.logger.Warnf("Identical existing for ImageCode (%s)\n Skipping later one...", imageCode)
			continue
		}
		m.baseImageMetaList[imageCode] = GetBaseImageMeta(&datavolume)
	}
	return nil
}

// This function can be used by gRPC call with update on DataVolume.
func (m *ImageManager) UpdateBaseImageList() error {
	return nil
}

// Refer BaseImageMeta with ImageCode.
func (m *ImageManager) ReferBaseImageMeta(imageCode VirtualMachineImageCode) (*BaseImageMeta, bool) {
	baseImageMeta, ok := m.baseImageMetaList[imageCode]
	return baseImageMeta, ok
}

func (m *ImageManager) listDataVolumeByLabel(label map[string]string) (*cdiv1alpha1.DataVolumeList, error) {
	options := utils.GetListOptionsWithLabel(label)
	return m.CdiV1alpha1().DataVolumes(corev1.NamespaceAll).List(options)
}

type VMWorkloadManager struct {
	preset *defaultVirtualMachine
	config *utils.KauloudConfig
}

func NewVMWorkloadManager(config *utils.KauloudConfig) *VMWorkloadManager {
	preset := DefaultVirtualMachine(ParseDefaultVmOptions(config))
	return &VMWorkloadManager{
		preset: preset,
		config: config,
	}
}

func (o *VMWorkloadManager) PresetVirtualMachine() *defaultVirtualMachine {
	return o.preset
}

func (o *VMWorkloadManager) NewVmWorkload(template *ParsedVmCreationTemplate) *VirtualMachineWorkload {
	new := &VirtualMachineWorkload{
		VM:       o.NewVirtualMachineObject(template),
		NodePort: nil,
	}
	new.attachServices()
	return new
}

func (o *VMWorkloadManager) NewVirtualMachineObject(template *ParsedVmCreationTemplate) *virtv1.VirtualMachine {
	must := map[string]string {
		kauloud.LabelKeyKauloudUserID:    template.UserID,
		kauloud.LabelKeyKauloudImageCode: template.ImageCode.String(),
		kauloud.LabelKeyKauloudUUID:      template.UUID.String(),
	}
	builder := o.PresetVirtualMachine().GetVirtualMachineBuilder()
	return builder.Config(o.config).Template(template).Labels(must).Build()
}

type VirtualMachineWorkload struct {
	VM        *virtv1.VirtualMachine
	NodePort  *corev1.Service
	ClusterIP *corev1.Service
}

func (o *VirtualMachineWorkload) attachServices() error {
	if o.VM == nil || o.VM.Labels == nil {
		return errors.New("VirtualMachineWorkload object is nil or Labels are not initialized")
	}
	userid, ok := o.VM.Labels[kauloud.LabelKeyKauloudUserID]
	if !ok {
		return errors.New("no matching label key for kauloud.LabelKeyKauloudUserID")
	}

	uuid, ok := o.VM.Labels[kauloud.LabelKeyKauloudUUID]
	if !ok {
		return errors.New("no matching label key for kauloud.LabelKeyKauloudUUID")
	}
	serviceLabel := map[string]string{
		kauloud.LabelKeyKauloudUserID: userid,
		kauloud.LabelKeyKauloudUUID:   uuid,
	}

	// set default sshd port
	ports := []corev1.ServicePort{
		{
			Name: "ssh",
			Port: 22,
			TargetPort: intstr.FromInt(22),
			Protocol: "TCP",
		},
	}

	o.NodePort = &corev1.Service{
		TypeMeta:   metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "corev1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:                       utils.NameServiceOnKubeResource(o.VM.Name, "NodePort"),
			Namespace:                  o.VM.Namespace,
			Labels:						serviceLabel,
		},
		Spec:       corev1.ServiceSpec{
			Ports:                    ports,
			Selector:                 serviceLabel,
			Type:                     kauloud.NodePortServiceType,
		},
		Status: corev1.ServiceStatus{},
	}

	o.ClusterIP = &corev1.Service{
		TypeMeta:   metav1.TypeMeta{
			Kind: "Service",
			APIVersion: "corev1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:                       utils.NameServiceOnKubeResource(o.VM.Name, "ClusterIP"),
			Namespace:                  o.VM.Namespace,
			Labels:                     serviceLabel,
		},
		Spec:       corev1.ServiceSpec{
			Ports:                    ports,
			Selector:                 serviceLabel,
			Type:                     kauloud.ClusterIPServiceType,
		},
		Status:     corev1.ServiceStatus{},
	}
	return nil
}

type InstanceTypeLister struct {
	InstanceTypeList map[InstanceTypeCode]InstanceType
}

func NewInstanceTypeLister(defaults ...InstanceType) *InstanceTypeLister {
	list := make(map[InstanceTypeCode]InstanceType)
	new := &InstanceTypeLister{
		InstanceTypeList: list,
	}
	new.init(defaults...)
	return new
}

func (o *InstanceTypeLister) init(defaults ...InstanceType) {
	for _, instance := range defaults {
		o.InstanceTypeList[instance.Code] = instance
	}
}

func (o *InstanceTypeLister) AddInstanceType(new InstanceType) error {
	if !new.IsValid() {
		return errors.New("")
	}
	o.InstanceTypeList[new.Code] = new
	return nil
}

func (o *InstanceTypeLister) DeleteInstanceType(code InstanceTypeCode) {
	delete(o.InstanceTypeList, code)
}

func (o *InstanceTypeLister) GetInstanceType(code InstanceTypeCode) (InstanceType, bool) {
	instanceType, exists := o.InstanceTypeList[code]
	return instanceType, exists
}

func DefaultInstanceTypes () []InstanceType {


	micro := InstanceType{
		Code:      1 ,
		Name:      "InstanceType-CPU-Micro",
		Resources: kauloud.Resources{
			Cpu:    "1",
			Memory: "2Gi",
			Gpu:    kauloud.Gpu{
				Enabled:    false,
				DeviceName: "",
			},
		},
	}

	c24 := InstanceType{
		Code:      2 ,
		Name:      "InstanceType-CPU-c24",
		Resources: kauloud.Resources{
			Cpu:    "2",
			Memory: "4Gi",
			Gpu:    kauloud.Gpu{
				Enabled:    false,
				DeviceName: "",
			},
		},
	}

	c48 := InstanceType{
		Code:      3 ,
		Name:      "InstanceType-CPU-c48",
		Resources: kauloud.Resources{
			Cpu:    "4",
			Memory: "8Gi",
			Gpu:    kauloud.Gpu{
				Enabled:    false,
				DeviceName: "",
			},
		},
	}

	g48 := InstanceType{
		Code:      4 ,
		Name:      "InstanceType-GPU-g48",
		Resources: kauloud.Resources{
			Cpu:    "4",
			Memory: "8Gi",
			Gpu:    kauloud.Gpu{
				Enabled:    true,
			},
		},
	}

	defaults := []InstanceType{
		micro, c24, c48, g48,
	}

	return defaults
}
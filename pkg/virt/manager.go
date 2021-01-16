package virt

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/xjayleex/kauloud/pkg/api/kauloud"
	"github.com/xjayleex/kauloud/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	virtv1 "kubevirt.io/client-go/api/v1"
	cdiclient "kubevirt.io/client-go/generated/containerized-data-importer/clientset/versioned"
	cdiv1alpha1core "kubevirt.io/client-go/generated/containerized-data-importer/clientset/versioned/typed/core/v1alpha1"
	"kubevirt.io/client-go/kubecli"
	cdiv1alpha1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
)

type VirtManager struct {
	config      *utils.KauloudConfig
	logger *logrus.Logger

	virtClient  kubecli.KubevirtClient
	workloadMgr *VmWorkloadManager
	imageMgr    *ImageManager
}

func NewVirtManager (config *utils.KauloudConfig, logger *logrus.Logger, virtClient kubecli.KubevirtClient) (*VirtManager, error){
	// TODO :: Client Config from outside args
	new := &VirtManager{
		config: config,
		logger: logger,
		virtClient: virtClient,
	}

	imageMgr, err := NewImageManager(new.VirtClient().CdiClient(), config)
	workloadMgr := NewVmWorkloadManager(new.config)

	new.imageMgr = imageMgr
	new.workloadMgr = workloadMgr

	return new, err
}

func (o *VirtManager) Logger() *logrus.Logger {
	return o.logger
}

func (o *VirtManager) VirtClient() kubecli.KubevirtClient {
	return o.virtClient
}

func (o *VirtManager) ImageMgr() *ImageManager {
	return o.imageMgr
}

func (o *VirtManager) WorkloadMgr() *VmWorkloadManager {
	return o.workloadMgr
}

func (o *VirtManager) ListVmi(namespace string, options *metav1.ListOptions) (*virtv1.VirtualMachineInstanceList, error) {
	vmiList, err := o.VirtClient().VirtualMachineInstance(namespace).List(options)
	return vmiList, err
}

func (o *VirtManager) CreateVmWorkload (req VmCreationTemplate) error {
	imageMeta, ok := o.ImageMgr().RefBaseImageMetaMap(req.ImageCode)
	if !ok {
		return errors.New("no matching images")
	}

	template := &ParsedVmCreationTemplate{
		VmCreationTemplate: &req,
		BaseImageMeta:      imageMeta,
		UUID:               uuid.New(),
	}
	workload := o.WorkloadMgr().NewVmWorkload(template)
	svc, err := o.VirtClient().CoreV1().Services(o.config.VirtManagerConfig.Namespace).Create(workload.Service)
	if err != nil {
		return err
	}

	_, err = o.VirtClient().VirtualMachine(o.config.VirtManagerConfig.Namespace).Create(workload.VirtualMachine)
	if err != nil {
		fmt.Println(err)
		err = o.VirtClient().CoreV1().Services(o.config.VirtManagerConfig.Namespace).Delete(svc.Name, &metav1.DeleteOptions{})
		return err
	}
	return err
}

type ImageManager struct {
	cdiclient.Interface
	config           *utils.KauloudConfig
	imageCodes		 []VmImageCode
	baseImageMetaMap map[VmImageCode]*BaseImageMeta
}

func NewImageManager (cdiclient cdiclient.Interface, config *utils.KauloudConfig) (*ImageManager, error) {
	imageMgr := &ImageManager{
		Interface: cdiclient,
		config:    config,
	}

	err := imageMgr.loadBaseImageMeta()

	return imageMgr, err
}

func (o *ImageManager) loadBaseImageMeta() error {
	baseImageMap := map[VmImageCode]*BaseImageMeta{}
	baseImages, err := o.baseImages()
	if err != nil {
		return err
	}
	for _, e := range baseImages.Items {
		baseImageMap[VmImageCode(e.Labels[kauloud.LabelKeyKauloudImageCode])] = GetBaseImageMeta(&e)
	}
	o.baseImageMetaMap = baseImageMap
	return err
}

func (o *ImageManager) RefBaseImageMetaMap(code VmImageCode) (*BaseImageMeta, bool) {
	baseImageMeta, ok := o.baseImageMetaMap[code]
	return baseImageMeta, ok
}

func (o *ImageManager) ListDataVolume(namespace string, options metav1.ListOptions) (*cdiv1alpha1.DataVolumeList, error) {
	dvList, err := o.CdiV1alpha1().DataVolumes(namespace).List(options)
	return dvList, err
}

func (o *ImageManager) ListDataVolumesByLabel(namespace string, labelSet map[string]string)  (*cdiv1alpha1.DataVolumeList, error) {
	listOptions := utils.GetListOptionsWithLabel(labelSet)
	dvList, err := o.ListDataVolume(namespace, listOptions)
	return dvList, err
}

func (o *ImageManager) baseImageDataVolume() cdiv1alpha1core.DataVolumeInterface {
	return o.CdiV1alpha1().DataVolumes(o.config.VirtManagerConfig.BaseImageNamespace)
}

func (o *ImageManager) ListBaseImageDataVolume(options metav1.ListOptions) (*cdiv1alpha1.DataVolumeList, error) {
	return o.baseImageDataVolume().List(options)
}

func (o *ImageManager) ListBaseImageDataVolumeByLabel (labelSet map[string]string) (*cdiv1alpha1.DataVolumeList, error) {
	listOptions := getListOptionsWithLabel(labelSet)
	return o.baseImageDataVolume().List(listOptions)
}

func (o *ImageManager) baseImages() (*cdiv1alpha1.DataVolumeList, error){
	labelSet := map[string]string {
		kauloud.LabelKeyKauloudDvType: kauloud.LabelValueKauloudDvTypeBaseImage,
	}
	return o.ListBaseImageDataVolumeByLabel(labelSet)
}

func (o *ImageManager) baseImagesByImageCode(code VmImageCode) (*cdiv1alpha1.DataVolume, error) {
	labelSet := map[string]string {
		kauloud.LabelKeyKauloudDvType: kauloud.LabelValueKauloudDvTypeBaseImage,
		kauloud.LabelKeyKauloudImageCode: code.String(),
	}
	dvList, err := o.ListBaseImageDataVolumeByLabel(labelSet)

	if err != nil {
		return nil, err
	}

	if len(dvList.Items) <= 0 {
		return nil, errors.New("no matching base image")
	} else if len(dvList.Items) > 1 {
		return nil, errors.New("no unique base image")
	}

	return &dvList.Items[0], nil
}

func (o *ImageManager) getBaseImageAnnotation(code VmImageCode) (map[string]string, error) {
	if dv, err := o.baseImagesByImageCode(code); err != nil {
		return nil, err
	} else {
		return dv.Annotations, err
	}
}

func (o *ImageManager) UpdatedBaseImageMeta(code VmImageCode) (*ImageOsMeta, error) {
	annotation, err := o.getBaseImageAnnotation(code)
	if err != nil {
		return nil, err
	}
	imageMeta := GetImageOsMetaFromAnnotation(annotation)
	return imageMeta, nil
}

type VmWorkloadManager struct {
	vmPreset *defaultVirtualMachine
	config *utils.KauloudConfig
}

func NewVmWorkloadManager(config *utils.KauloudConfig) *VmWorkloadManager {
	vmPreset := DefaultVirtualMachine(ParseDefaultVmOptions(config))
	return &VmWorkloadManager{
		vmPreset: vmPreset,
		config:   config,
	}
}

func (o *VmWorkloadManager) VmPreset() *defaultVirtualMachine {
	return o.vmPreset
}

func (o *VmWorkloadManager) NewVmWorkload(template *ParsedVmCreationTemplate) *VmWorkload {
	new := &VmWorkload{
		VirtualMachine: o.newVmObject(template),
		Service:        nil,
	}
	new.attachService()
	return new
}

func (o *VmWorkloadManager) newVmObject(template *ParsedVmCreationTemplate) *virtv1.VirtualMachine {
	must := map[string]string {
		kauloud.LabelKeyKauloudUserId:    template.UserId,
		kauloud.LabelKeyKauloudImageCode: template.ImageCode.String(),
		kauloud.LabelKeyKauloudUUID:      template.UUID.String(),
	}
	builder := o.VmPreset().GetNewVmBuilder()

	return builder.Config(o.config).Template(template).Labels(must).Build()
}

type VmWorkload struct {
	*virtv1.VirtualMachine
	*corev1.Service
}

func (o *VmWorkload) attachService(additional ...corev1.ServicePort) (*VmWorkload, error) {
	// TODO : Vm 생성 & Svc 생성
	vmUUID, ok := o.VirtualMachine.Labels[kauloud.LabelKeyKauloudUUID]
	if !ok {
		return nil, errors.New("no matching vm UUID")
	}
	serviceLabel := map[string]string{
		kauloud.LabelKeyKauloudUUID: vmUUID,
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

	ports = append(ports, additional...)

	svc := &corev1.Service{
		TypeMeta:   metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "corev1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:                       fmt.Sprintf("%s-svc", o.VirtualMachine.Name),
			Namespace:                  o.VirtualMachine.Namespace,
		},
		Spec:       corev1.ServiceSpec{
			Ports:                    ports,
			Selector:                 serviceLabel,
			Type:                     "NodePort",
		},
		Status: corev1.ServiceStatus{},
	}

	o.Service = svc
	return o, nil
}


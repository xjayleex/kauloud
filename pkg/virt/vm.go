package virt

import (
	"fmt"
	"github.com/jinzhu/copier"
	"github.com/xjayleex/kauloud/pkg/api/kauloud"
	"github.com/xjayleex/kauloud/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	virtv1 "kubevirt.io/client-go/api/v1"
	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
)


type DefaultVmOptions struct {
	Namespace                 string
	RunStrategy               virtv1.VirtualMachineRunStrategy
	DomainCpuCores            uint32
	CpuRequest, MemoryRequest string
	CpuLimit, MemoryLimit     string
	StorageRequest            string
	BaseImageNamespace        string
	BaseImageName             string
	Labels                    map[string]string
	StorageClassName          string

	useKVMHiddenState            bool
	accessMode                   []corev1.PersistentVolumeAccessMode
	bootVolumeInterfaceName      string
	timezone                     virtv1.ClockOffsetTimezone
	cloudInitVolumeInterfaceName string
}

func ParseDefaultVmOptions(config *utils.KauloudConfig) *DefaultVmOptions {
	opts := &DefaultVmOptions{}
	virtConfig := &config.VirtManagerConfig
	opts.Namespace = virtConfig.Namespace
	opts.RunStrategy = virtv1.VirtualMachineRunStrategy(virtConfig.VmPreset.RunStrategy)
	opts.DomainCpuCores = virtConfig.VmPreset.DomainCpuCores
	opts.CpuRequest, opts.CpuLimit = virtConfig.VmPreset.Resources.Cpu, virtConfig.VmPreset.Resources.Cpu
	opts.MemoryRequest, opts.MemoryLimit = virtConfig.VmPreset.Resources.Memory, virtConfig.VmPreset.Resources.Memory
	opts.StorageRequest = virtConfig.VmPreset.Resources.Storage
	opts.BaseImageNamespace = virtConfig.BaseImageNamespace
	opts.BaseImageName = virtConfig.VmPreset.PresetBaseImageName

	// Settings below are can be hardcoded.
	opts.StorageClassName = virtConfig.StorageClassName
	opts.Labels = virtConfig.VmPreset.Labels

	opts.useKVMHiddenState = true
	opts.accessMode = append(opts.accessMode, "ReadWriteOnce")
	opts.timezone = virtv1.ClockOffsetTimezone("Asia/Seoul")
	opts.bootVolumeInterfaceName = "root"
	opts.cloudInitVolumeInterfaceName = "cloudinitvolume"
	return opts
}

type defaultVirtualMachine struct {
	VirtualMachine *virtv1.VirtualMachine
}

func DefaultVirtualMachine(opts *DefaultVmOptions) *defaultVirtualMachine {

	return &defaultVirtualMachine{
		VirtualMachine: &virtv1.VirtualMachine{
			TypeMeta:   metav1.TypeMeta{
				Kind:       "VM",
				APIVersion: "kubevirt.io/v1alpha3",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: opts.Namespace,
				Labels: opts.Labels,
			},
			Spec: virtv1.VirtualMachineSpec{
				RunStrategy:         &opts.RunStrategy,
				Template:            &virtv1.VirtualMachineInstanceTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: opts.Labels,
					},
					Spec:       virtv1.VirtualMachineInstanceSpec{
						Domain: virtv1.DomainSpec{
							Features: &virtv1.Features{
								KVM:    &virtv1.FeatureKVM{
									Hidden: opts.useKVMHiddenState,
								},
							},
							CPU: &virtv1.CPU{
								Cores:                 opts.DomainCpuCores,
							},
							Resources: utils.GetVirtResourceRequirements(utils.GetCpuMemoryResourceList(opts.CpuRequest, opts.MemoryRequest), utils.GetCpuMemoryResourceList(opts.CpuLimit, opts.MemoryLimit)),
							Devices: virtv1.Devices{
								Disks:                      nil,
								Interfaces:                 []virtv1.Interface{
									{
										Name: "default",
										InterfaceBindingMethod: virtv1.InterfaceBindingMethod{
											Bridge: &virtv1.InterfaceBridge{},
										},
										MacAddress: utils.NewUniCastMacAddress(),
									},
								},
							},
							Clock: &virtv1.Clock{
								ClockOffset: virtv1.ClockOffset{
									Timezone: &opts.timezone,
								},
								Timer: &virtv1.Timer{
									KVM:    &virtv1.KVMTimer{
										Enabled: utils.NewTrue(),
									},
								},
							},
							Machine: virtv1.Machine{ Type: "" },
						},
						Networks: []virtv1.Network{
							{
								Name: "default",
								NetworkSource: virtv1.NetworkSource{
									Pod: &virtv1.PodNetwork{},
								},
							},
						},
						Volumes: []virtv1.Volume{
							{
								Name: opts.bootVolumeInterfaceName,
								VolumeSource: virtv1.VolumeSource{
									DataVolume: &virtv1.DataVolumeSource{
									},
								},
							},
							{
								Name: opts.cloudInitVolumeInterfaceName,
								VolumeSource: virtv1.VolumeSource{
									CloudInitNoCloud: &virtv1.CloudInitNoCloudSource{
										UserDataBase64: kauloud.DefaultUserDataBase64,
									},
								},
							},
						},
					},
				},
				DataVolumeTemplates: []virtv1.DataVolumeTemplateSpec{
					{
						ObjectMeta: metav1.ObjectMeta{
						},
						Spec: cdiv1.DataVolumeSpec{
							PVC: &corev1.PersistentVolumeClaimSpec{
							},
							Source: cdiv1.DataVolumeSource{
								PVC: &cdiv1.DataVolumeSourcePVC{
								},
							},
							ContentType: "",
						},
						Status: &virtv1.DataVolumeTemplateDummyStatus{},
					},
				},
			},
			Status: virtv1.VirtualMachineStatus{},
		},
	}
}

func (o *defaultVirtualMachine) GetVirtualMachineBuilder() *virtualMachineBuilder {
	new := &virtv1.VirtualMachine{}
	copier.Copy(new, o.VirtualMachine)
	return &virtualMachineBuilder{ new: new }
}

type virtualMachineBuilder struct {
	new *virtv1.VirtualMachine

	config *utils.KauloudConfig
	template *ParsedVmCreationTemplate
	labels map[string]string
}

func (b *virtualMachineBuilder) Template(template *ParsedVmCreationTemplate) *virtualMachineBuilder {
	b.template = template
	return b
}

func (b *virtualMachineBuilder) Labels(labels map[string]string) *virtualMachineBuilder {
	b.labels = labels
	return b
}

func (b *virtualMachineBuilder) Config(config *utils.KauloudConfig) *virtualMachineBuilder {
	b.config = config
	return b
}

func (b *virtualMachineBuilder) Build() *virtv1.VirtualMachine {
	nilString := ""
	fmt.Println(b.config.VirtManagerConfig.StorageClassName)
	if b.template.Resources.Gpu.Enabled {
		b.new.Spec.Template.Spec.Domain.Devices.GPUs = []virtv1.GPU{
			{
				Name: "gpu",
				DeviceName: b.template.Resources.Gpu.DeviceName,
			},
		}
	}

	namingFunc := func(prefix string) string{
		return fmt.Sprintf("%s-%s-%s", prefix, b.template.UserID, utils.GetTimeLowFromUUID(b.template.UUID))
	}

	b.new.ObjectMeta.Name = namingFunc(kauloud.PrefixNameVirtualMachine)

	b.new.ObjectMeta.Labels, b.new.Spec.Template.ObjectMeta.Labels = b.labels, b.labels

	b.new.Spec.Template.Spec.Domain.CPU.Cores = utils.StringAsUint32(b.template.Resources.Cpu)
	b.new.Spec.Template.Spec.Domain.Resources = utils.GetVirtResourceRequirements(utils.GetCpuMemoryResourceList(b.template.Resources.Cpu, b.template.Resources.Memory),utils.GetCpuMemoryResourceList(b.template.Resources.Cpu, b.template.Resources.Memory))
	b.new.Spec.Template.Spec.Domain.Devices.Interfaces[0].MacAddress = utils.NewUniCastMacAddress()
	b.new.Spec.Template.Spec.Domain.Devices.Disks = []virtv1.Disk{
		{
			Name: "boot",
			DiskDevice:  virtv1.DiskDevice{
				Disk: &virtv1.DiskTarget{
					Bus: "virtio",
				},
			},
		},
		{
			Name: "cloudinitvolume",
			DiskDevice: virtv1.DiskDevice{
				Disk: &virtv1.DiskTarget{
					Bus: "virtio",
				},
			},
		},
	}
	b.new.Spec.Template.Spec.Volumes = []virtv1.Volume {
		{
			Name: "boot",
			VolumeSource: virtv1.VolumeSource{
				DataVolume: &virtv1.DataVolumeSource{
					Name: namingFunc(kauloud.PrefixNameDataVolume),
				},
			},
		},
		{
			Name: "cloudinitvolume",
			VolumeSource: virtv1.VolumeSource{
				CloudInitNoCloud: &virtv1.CloudInitNoCloudSource{
					UserDataBase64: kauloud.DefaultUserDataBase64,
				},
			},
		},
	}

	// labeling datavolume to notify it is for user virtual machine datavolume.
	b.labels[kauloud.LabelKeyKauloudDvType] = kauloud.LabelValueKauloudDvTypeUserVmImage
	b.new.Spec.DataVolumeTemplates = []virtv1.DataVolumeTemplateSpec {
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: namingFunc(kauloud.PrefixNameDataVolume),
				Labels: b.labels,
			},
			Spec: cdiv1.DataVolumeSpec{
				PVC: &corev1.PersistentVolumeClaimSpec{
					StorageClassName: &b.config.VirtManagerConfig.StorageClassName,
					AccessModes: []corev1.PersistentVolumeAccessMode{
						"ReadWriteOnce",
					},
					Resources: utils.GetKubeResourceRequirements(utils.GetStorageResourceList(b.template.Size), utils.GetStorageResourceList("")),
					DataSource: &corev1.TypedLocalObjectReference{
						APIGroup: &nilString,
					},
				},
				Source: cdiv1.DataVolumeSource{
					PVC: &cdiv1.DataVolumeSourcePVC{
						Name: b.template.Name,
						Namespace: b.config.VirtManagerConfig.BaseImageNamespace,
					},
				},
			},
		},
	}

	return b.new
}
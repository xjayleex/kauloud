package virt

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/xjayleex/kauloud/pkg/api/kauloud"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	cdiv1alpha1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
)
type VmCreationTemplate struct {
	UserID    string
	ImageCode VirtualMachineImageCode
	Resources kauloud.Resources
}

type ParsedVmCreationTemplate struct {
	*VmCreationTemplate
	*BaseImageMeta
	UUID               uuid.UUID
}

type BaseImageMeta struct{
	Name   string
	OsMeta *ImageOsMeta
	Size   resource.Quantity
}

func GetBaseImageMeta(datavolume *cdiv1alpha1.DataVolume) *BaseImageMeta {
	quantity, _ := datavolume.Spec.PVC.Resources.Requests[corev1.ResourceStorage]
	return &BaseImageMeta{
		Name:   datavolume.Name,
		OsMeta: GetImageOsMetaFromAnnotation(datavolume.GetAnnotations()),
		Size:   quantity,
	}
}

type ImageOsMeta struct {
	OsType    string
	OsVersion string
}

func GetImageOsMetaFromAnnotation(annotation map[string]string) *ImageOsMeta {
	osType, ok := annotation[kauloud.AnnotationKeyOsType]
	if !ok {
		osType = "unknown"
	}

	osVersion, ok := annotation[kauloud.AnnotationKeyOsVersion]
	if !ok {
		osType = "unknown"
	}

	return &ImageOsMeta{
		OsType: osType,
		OsVersion: osVersion,
	}
}

func (o *ImageOsMeta) String() string {
	return fmt.Sprintf("%s-%s", o.OsType, o.OsVersion)
}

type VirtualMachineImageCode string

func (s VirtualMachineImageCode) String () string {
	return string(s)
}

// Instance Type

type InstanceType struct {
	Code		InstanceTypeCode
	Name 		INstanceTypeName
	Resources	kauloud.Resources
}

func (t *InstanceType) IsValid() bool {
	if t.Code <= 0 || t.Name == "" {
		return false
	}
	// Todo :: InstanceType.Resource Validation
	return true
}

type InstanceTypeCode int32
type INstanceTypeName string
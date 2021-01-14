package virt

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/xjayleex/kauloud/pkg/api/kauloud"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	cdiv1alpha1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
)
type ReqFromGW struct {
	UserId    string
	ImageCode VmImageCode
	Resources Resources
	Gpu       Gpu
}

type VmCreationTemplate struct {
	*ReqFromGW
	*BaseImageMeta
	UUID uuid.UUID
}

type BaseImageMeta struct{
	ImageName   string
	ImageOsMeta *ImageOsMeta
	ImageSize   resource.Quantity
}

func GetBaseImageMeta(dv *cdiv1alpha1.DataVolume) *BaseImageMeta {
	quantity, _ := dv.Spec.PVC.Resources.Requests[corev1.ResourceStorage]
	return &BaseImageMeta{
		ImageName:   dv.Name,
		ImageOsMeta: GetImageOsMetaFromAnnotation(dv.GetAnnotations()),
		ImageSize:   quantity,
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

type Gpu struct {
	Enabled    bool
	DeviceName string
}

type Resources struct {
	Cpu string
	Memory string
}

type VmImageCode string

func (s VmImageCode) String () string {
	return string(s)
}
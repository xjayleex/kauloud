package kauloud

import corev1 "k8s.io/api/core/v1"

const (
	// Label assets
	LabelKeyKauloudUserID    = "kauloud/userid"
	LabelKeyKauloudDvType    = "kauloud/dvtype"
	LabelKeyKauloudImageCode = "kauloud/imagecode"
	LabelKeyKauloudUUID      = "kauloud/uuid"

	LabelValueKauloudDvTypeBaseImage   = "base-image"
	LabelValueKauloudDvTypeUserVmImage = "vm-image"

	// Annotation assets
	AnnotationKeyOsType = "OsType"
	AnnotationKeyOsVersion = "OsVersion"
	AnnotationKeyImageSize = "Size"

	PrefixNameDataVolume = "dv"
	PrefixNameVirtualMachine = "vm"
	PrefixNameService = "svc"

	ResourceAbbrDataVolume     = "dv"
	ResourceAbbrVirtualMachine = "vm"
	ResourceAbbrPod            = "pod"
	ResourceAbbrService        = "svc"
	ResourceAbbrNode        = "node"

	WatcherEventVerbAdd    = "add"
	WatcherEventVerbDelete = "delete"
	WatcherEventVerbUpdate = "update"


	// belows are needed to be sorted.
	DefaultUserDataBase64 = "I2Nsb3VkLWNvbmZpZw0KcGFzc3dvcmQ6IHVidW50dQ0Kc3NoX3B3YXV0aDogVHJ1ZQ0KY2hwYXNzd2Q6IHsgZXhwaXJlOiBGYWxzZSB9"
	ControllerMaxRequeue = 5
	TargetNamespace      = corev1.NamespaceDefault
	AllNamespace = corev1.NamespaceAll
	DataVolumeSucceed    = "Succeeded"
	DataVolumeDeleted	 = "DataVolumeDeleted"
	None = "None"

	DefaultInformerThreadiness = 1

	NodePortServiceType = "NodePort"
	ClusterIPServiceType = "ClusterIP"

	AnonymousUser = "Anonymous"
)
package kauloud

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// Define commonly used types in our kauloud system.

type ResourceSetter interface {
	Cpu () string
	Memory () string
	Gpu () Gpu
}

// Todo :: Would this struct should implement ResourceSetter?
type Resources struct {
	Cpu string
	Memory string
	Gpu	Gpu
}

func NewResourceFromSetter(setter ResourceSetter) *Resources {
	return &Resources{
		Cpu: setter.Cpu(),
		Memory: setter.Memory(),
		Gpu: setter.Gpu(),
	}
}

// if MustParse() can occur Panic, must implement hook to check resource string field in Resources{}
// but it should be easier to use ParseQuantity() to check such an error.
func (r *Resources) ParseResourceList () (corev1.ResourceList, error) {
	cpuQuantity, err := resource.ParseQuantity(r.Cpu)
	if err != nil {
		return nil, err
	}
	memoryQuantity, err := resource.ParseQuantity(r.Memory)
	if err != nil {
		return nil, err
	}
	gpuQuantity, err := resource.ParseQuantity("1")
	if err != nil {
		return nil, err
	}

	parsed := corev1.ResourceList{
		"cpu": cpuQuantity,
		"memory": memoryQuantity,
		corev1.ResourceName(r.Gpu.DeviceName): gpuQuantity,
	}

	return parsed, nil
}

type Gpu struct {
	Enabled    bool
	DeviceName string
}

type UUID string

func (u *UUID) String() string {
	return string(*u)
}

type UserID string

func (u *UserID) String() string {
	return string(*u)
}

// Store workload metadata, owner and uuid.
type WorkloadMeta struct {
	Owner UserID
	UUID  UUID
}
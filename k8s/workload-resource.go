package main

import (
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PodObjectBuilder interface {
	Build()									*core.Pod
	SetObjectMeta(m metav1.ObjectMeta)		PodObjectBuilder
	SetSpec(s core.PodSpec)					PodObjectBuilder
}

type podObject struct {
	meta	metav1.ObjectMeta
	spec	core.PodSpec
}

func (po *podObject) SetObjectMeta (m metav1.ObjectMeta) {
	po.meta = m
}

func (po *podObject) SetSpec(s core.PodSpec) {
	po.spec = s
}

func (po *podObject) Build() *core.Pod {
	return &core.Pod{
		ObjectMeta: po.meta,
		Spec: po.spec,
	}
}

type DefaultPodObject struct {
	*podObject
}

func (d *DefaultPodObject) Build () *core.Pod {
	var hostPathType core.HostPathType
	hostPathType = "Directory"

	// Set Object Metadata.
	d.SetObjectMeta(metav1.ObjectMeta{
		Name: "linux-cent-dev",
		Namespace: "dev",
		Labels: map[string]string {
			"app": "demo",
		},
	})

	d.SetSpec(core.PodSpec{
		NodeName: "dn2",
		Containers: []core.Container{
			{
				Name: "cent-dev",
				Image: "cent7:7",
				ImagePullPolicy: core.PullIfNotPresent,
				Command: []string {
					"init",
					"/bin/bash",
					"-c",
					"while true; do sleep 1000; done",
				},
				SecurityContext: &core.SecurityContext{
					Capabilities:             nil,
					Privileged:               nil,
					SELinuxOptions:           nil,
					WindowsOptions:           nil,
					RunAsUser:                nil,
					RunAsGroup:               nil,
					RunAsNonRoot:             nil,
					ReadOnlyRootFilesystem:   nil,
					AllowPrivilegeEscalation: nil,
					ProcMount:                nil,
				},
				VolumeMounts: []core.VolumeMount{
					{
						Name:             "",
						ReadOnly:         false,
						MountPath:        "",
						SubPath:          "",
						MountPropagation: nil,
						SubPathExpr:      "",
					},
				},
			},
		},
		Volumes: []core.Volume{
			{
				Name:         "data",
				VolumeSource: core.VolumeSource{
					HostPath: &core.HostPathVolumeSource{
						Path: "/data/hdd/scratch",
						Type: &hostPathType,
					},
				},
			},
		},
	})

	return &core.Pod{
		ObjectMeta: d.meta,
		Spec:       d.spec,
	}
}
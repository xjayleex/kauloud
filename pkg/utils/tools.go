package utils

import (
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/xjayleex/kauloud/pkg/api/kauloud"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	virtv1 "kubevirt.io/client-go/api/v1"
	"strconv"
	"strings"
)

func GetListOptionsWithLabel(labelSet map[string]string) metav1.ListOptions {
	return metav1.ListOptions{LabelSelector: labels.SelectorFromSet(labelSet).String()}
}

func GetTimeLowFromUUID	(uuid uuid.UUID) string {
	return strings.Split(uuid.String(),"-")[0]
}

func NewTrue() *bool {
	new := true
	return &new
}
func GetCpuMemoryResourceList(cpu, memory string) corev1.ResourceList {
	res := corev1.ResourceList{}
	if cpu != "" {
		res[corev1.ResourceCPU] = resource.MustParse(cpu)
	}
	if memory != "" {
		res[corev1.ResourceMemory] = resource.MustParse(memory)
	}
	return res
}

func GetStorageResourceList(storage interface{}) corev1.ResourceList {

	res := corev1.ResourceList{}

	if quantity, ok := storage.(resource.Quantity); ok {
		res[corev1.ResourceStorage] = quantity
		return res
	}

	if quantity, ok := storage.(string); ok {
		res[corev1.ResourceStorage] = resource.MustParse(quantity)
	}

	return res
}

func GetVirtResourceRequirements(requests, limits corev1.ResourceList) virtv1.ResourceRequirements {
	res := virtv1.ResourceRequirements{}
	res.Requests = requests
	res.Limits = limits
	return res
}

func GetKubeResourceRequirements(requests, limits corev1.ResourceList) corev1.ResourceRequirements {
	res := corev1.ResourceRequirements{}
	res.Requests = requests
	res.Limits = limits
	return res
}

func NewUniCastMacAddress() string {
	buf := make([]byte, 6)
	_, err := rand.Read(buf)
	if err != nil {
		return NewUniCastMacAddress()
	}
	buf[0] &= buf[0] - 1
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", buf[0], buf[1], buf[2], buf[3], buf[4], buf[5])
}

func StringAsUint32(n string) uint32 {
	integer, _ := strconv.Atoi(n)
	return uint32(integer)
}

// do not modify
func NameServiceOnKubeResource(objectName string, serviceType string) string {
	return fmt.Sprintf("svc-%s-%s", objectName, serviceType)
}

// Service name can be populated with splitting service name by '-' delimeter and getting
// last element of tokenized tokens. It is associated with our mechanism to name the service.
// Refer the utils.NameServiceOnKubeResource() method.
func GetServiceTypeFromEventKey(key string) (corev1.ServiceType, error) {
	tokens := strings.Split(key,"-")
	serviceType := tokens[len(tokens)-1]
	switch serviceType {
	case strings.ToLower(string(corev1.ServiceTypeClusterIP)):
		return corev1.ServiceTypeClusterIP, nil
	case strings.ToLower(string(corev1.ServiceTypeNodePort)):
		return corev1.ServiceTypeNodePort, nil
	default:
		return "", errors.New("no matching service types")
	}
}

// Extract USERID and UUID of VM from labels. If the entity does not exists in label,
// we should give it a temporary userid, and we also generate temporary code for uuid.
func ExtractWorkloadMetaFromLabels(label map[string]string) *kauloud.WorkloadMeta {
	UserID, ok := label[kauloud.LabelKeyKauloudUserID]
	if !ok {
		UserID = kauloud.AnonymousUser
	}
	UUID, ok := label[kauloud.LabelKeyKauloudUUID]
	if !ok {
		UUID = uuid.New().String()
	}
	return &kauloud.WorkloadMeta{
		Owner: kauloud.UserID(UserID),
		UUID:  kauloud.UUID(UUID),
	}
}
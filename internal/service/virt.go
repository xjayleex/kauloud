package service

import (
	"context"
	"github.com/xjayleex/kauloud/pkg/virt"
	pb "github.com/xjayleex/kauloud/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type VirtManagementService struct {
	virtManager           *virt.Manager
	workloadStatusService pb.WorkloadStatusServiceClient
}

func NewVirtManagementService (virtManager *virt.Manager, workloadStatusClient pb.WorkloadStatusServiceClient) *VirtManagementService{
	return &VirtManagementService{
		virtManager:           virtManager,
		workloadStatusService: workloadStatusClient,
	}
}

func (v *VirtManagementService) CreateVirtualMachine(ctx context.Context, req *pb.VmCreationRequest) (*pb.DummyResponse, error) {
	// 1. Check Allocation Availability.
	workload := &pb.WorkloadInfo{
		Type:         0,
		UserId:       req.UserId,
		Token:        "",
		WorkloadName: "",
		Spec:         &pb.ContainerResourceSpec{
			Resources:   nil,
			ImageCode:   "",
			HostAliases: nil,
		},
	}
	v.workloadStatusService.IsAllocatable(ctx, workload)
	return nil, status.Errorf(codes.Unimplemented, "method CreateVirtualMachine not implemented")
}
func (v *VirtManagementService) DeleteVirtualMachine(context.Context, *pb.VmDeletionRequest) (*pb.DummyResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteVirtualMachine not implemented")
}

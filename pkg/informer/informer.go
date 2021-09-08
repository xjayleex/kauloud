package informer

import (
	"context"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/sirupsen/logrus"
	"github.com/xjayleex/kauloud/pkg/api/kauloud"
	"github.com/xjayleex/kauloud/pkg/utils"
	pb "github.com/xjayleex/kauloud/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type InformerService struct {
	logger			*logrus.Logger
	config			*utils.KauloudConfig
	gRPCServer		*grpc.Server

	watcher			*Watcher
	describer		ResourceDescriberInterface

}

func NewInformerService(config *utils.KauloudConfig, server *grpc.Server, logger *logrus.Logger, watcher *Watcher, describer ResourceDescriberInterface) *InformerService {
	return &InformerService{
		config: config,
		gRPCServer: server,
		logger: logger,
		watcher:   watcher,
		describer: describer,
	}
}


func (s *InformerService) Run (threadiness int, stopCh chan struct{}) {
	s.watcher.Run(threadiness, stopCh)
	s.describer.Run(threadiness, stopCh)
	<-stopCh
}

func (s *InformerService) ListAllVirtualMachineStats(ctx context.Context, req *empty.Empty) (*pb.VirtualMachineStats, error) {
	vmInfoList := s.watcher.VmInfoList()

	resp := &pb.VirtualMachineStats{
		VirtualMachineStatusMap: make(map[string]*pb.VirtualMachineStatus),
	}

	for _, list := range vmInfoList {
		for uuid, info := range list {
			resp.VirtualMachineStatusMap[uuid.String()] = info.Status
		}
	}

	if len(resp.VirtualMachineStatusMap) == 0 {
		return nil, status.Errorf(codes.NotFound, "no virtual machine exists")
	}

	return resp, nil
}

func (s *InformerService) ListUserVirtualMachineStats(ctx context.Context, userid *pb.UserID) (*pb.VirtualMachineStats, error) {
	list, err := s.watcher.GetUserVirtualMachineInfoList(kauloud.UserID(userid.GetUserid()))
	if err != nil {
		return nil, status.Errorf(codes.NotFound, err.Error())
	}

	resp := &pb.VirtualMachineStats{
		VirtualMachineStatusMap: make(map[string]*pb.VirtualMachineStatus),
	}

	for uuid, info := range list {
		resp.VirtualMachineStatusMap[uuid.String()] = info.Status
	}

	if len(resp.VirtualMachineStatusMap) == 0 {
		return nil, status.Errorf(codes.NotFound, "no virtual machine exists")
	}

	return resp, nil
}

func (s *InformerService) GetWorkloadObjectNameMeta(ctx context.Context, meta *pb.WorkloadMeta) (*pb.WorkloadObjectNameMeta, error) {
	info, err := s.watcher.vmInfoList.GetVirtualMachineInfo(kauloud.UserID(meta.Owner), kauloud.UUID(meta.UUID))
	if err != nil {
		return nil, status.Errorf(codes.NotFound, err.Error())
	}
	if info.Object.VirtualMachine() == nil || info.Object.ClusterIPService() == nil {
		return nil, status.Errorf(codes.Internal, "VirtualMachine or ClusterIPService doesn't exists")
	}

	resp := &pb.WorkloadObjectNameMeta{
		VirtualMachine: info.Object.VirtualMachine().Name,
		ClusterIp:      info.Object.ClusterIPService().Name,
	}

	return resp, nil
}

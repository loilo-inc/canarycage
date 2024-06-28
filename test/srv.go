package test

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/servicediscovery"
	srvtypes "github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
	"golang.org/x/xerrors"
)

type SrvServer struct {
	*commons
	services []*srvtypes.Service
	// service.Name -> []*instance
	insts map[string][]*srvtypes.Instance
	// instance.Id -> HealthStatus
	instHelths map[string]srvtypes.HealthStatus
}

func (s *SrvServer) getServiceById(id string) *srvtypes.Service {
	for _, svc := range s.services {
		if *svc.Id == id {
			return svc
		}
	}
	return nil
}

func (s *SrvServer) putInstHealth(id string, health srvtypes.HealthStatus) {
	s.instHelths[id] = health
}

func (s *SrvServer) DiscoverInstances(ctx context.Context, params *servicediscovery.DiscoverInstancesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.DiscoverInstancesOutput, error) {
	insts, ok := s.insts[*params.ServiceName]
	if !ok {
		return nil, xerrors.Errorf("service not found: %s", *params.ServiceName)
	}
	var summories []srvtypes.HttpInstanceSummary
	for _, inst := range insts {
		health := s.instHelths[*inst.Id]
		summories = append(summories, srvtypes.HttpInstanceSummary{
			Attributes:    inst.Attributes,
			InstanceId:    inst.Id,
			ServiceName:   params.ServiceName,
			NamespaceName: params.NamespaceName,
			HealthStatus:  health,
		})
	}
	return &servicediscovery.DiscoverInstancesOutput{Instances: summories}, nil
}

func (s *SrvServer) RegisterInstance(ctx context.Context, params *servicediscovery.RegisterInstanceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.RegisterInstanceOutput, error) {
	if srv := s.getServiceById(*params.ServiceId); srv == nil {
		return nil, xerrors.Errorf("service not found: %s", *params.ServiceId)
	} else {
		inst := &srvtypes.Instance{
			Id:         params.InstanceId,
			Attributes: params.Attributes,
		}
		s.insts[*srv.Name] = append(s.insts[*params.ServiceId], inst)
		s.instHelths[*params.InstanceId] = srvtypes.HealthStatusUnhealthy
		return &servicediscovery.RegisterInstanceOutput{}, nil
	}
}

func (s *SrvServer) DeregisterInstance(ctx context.Context, params *servicediscovery.DeregisterInstanceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.DeregisterInstanceOutput, error) {
	srv := s.getServiceById(*params.ServiceId)
	if srv == nil {
		return nil, xerrors.Errorf("service not found: %s", *params.ServiceId)
	}
	insts, ok := s.insts[*srv.Name]
	if !ok {
		return nil, xerrors.Errorf("service not found: %s", *srv.Name)
	}
	var newInsts []*srvtypes.Instance
	for _, inst := range insts {
		if *inst.Id != *params.InstanceId {
			newInsts = append(newInsts, inst)
		}
	}
	s.insts[*srv.Name] = newInsts
	return &servicediscovery.DeregisterInstanceOutput{}, nil
}

func (s *SrvServer) GetService(ctx context.Context, params *servicediscovery.GetServiceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.GetServiceOutput, error) {
	svc := s.getServiceById(*params.Id)
	if svc == nil {
		return nil, xerrors.Errorf("service not found: %s", *params.Id)
	}
	return &servicediscovery.GetServiceOutput{Service: svc}, nil
}

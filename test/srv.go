package test

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery"
	srvtypes "github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
	"golang.org/x/xerrors"
)

type SrvServer struct {
	*commons
}

func (s *SrvServer) getServiceById(id string) *srvtypes.Service {
	for _, svc := range s.SrvServices {
		if *svc.Id == id {
			return svc
		}
	}
	return nil
}

func (s *commons) CreateSrvService(
	namepsaceName string,
	serviceName string) *srvtypes.Service {
	nsId := fmt.Sprintf("ns-%s", namepsaceName)
	ns := &srvtypes.Namespace{
		Id:   &nsId,
		Name: &namepsaceName,
		Arn:  aws.String(fmt.Sprintf("arn:aws:servicediscovery:ap-northeast-1:123456789012:namespace/%s", nsId)),
	}
	svId := fmt.Sprintf("srv-%s", serviceName)
	svc := &srvtypes.Service{
		NamespaceId:   ns.Id,
		Id:            &svId,
		Name:          &serviceName,
		Arn:           aws.String(fmt.Sprintf("arn:aws:servicediscovery:ap-northeast-1:123456789012:service/%s", svId)),
		InstanceCount: aws.Int32(0),
	}
	s.SrvNamespaces = append(s.SrvNamespaces, ns)
	s.SrvServices = append(s.SrvServices, svc)
	return svc
}

func (s *commons) PutSrvInstHealth(id string, health srvtypes.HealthStatus) {
	s.SrvInstHelths[id] = health
}

func (s *SrvServer) DiscoverInstances(ctx context.Context, params *servicediscovery.DiscoverInstancesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.DiscoverInstancesOutput, error) {
	insts, ok := s.SrvInsts[*params.ServiceName]
	if !ok {
		return nil, xerrors.Errorf("service not found: %s", *params.ServiceName)
	}
	var summories []srvtypes.HttpInstanceSummary
	for _, inst := range insts {
		health := s.SrvInstHelths[*inst.Id]
		if !matchInst(inst, params) {
			continue
		}
		switch params.HealthStatus {
		case srvtypes.HealthStatusFilterHealthy:
			if health != srvtypes.HealthStatusHealthy {
				continue
			}
		case srvtypes.HealthStatusFilterUnhealthy:
			if health != srvtypes.HealthStatusUnhealthy {
				continue
			}
		}
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

func matchInst(inst *srvtypes.Instance, params *servicediscovery.DiscoverInstancesInput) bool {
	for k, v := range params.QueryParameters {
		if act, ok := inst.Attributes[k]; !ok || act != v {
			return false
		}
	}
	return true
}

func (s *SrvServer) RegisterInstance(ctx context.Context, params *servicediscovery.RegisterInstanceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.RegisterInstanceOutput, error) {
	if srv := s.getServiceById(*params.ServiceId); srv == nil {
		return nil, xerrors.Errorf("service not found: %s", *params.ServiceId)
	} else {
		inst := &srvtypes.Instance{
			Id:         params.InstanceId,
			Attributes: params.Attributes,
		}
		s.SrvInsts[*srv.Name] = append(s.SrvInsts[*params.ServiceId], inst)
		s.SrvInstHelths[*params.InstanceId] = srvtypes.HealthStatusUnhealthy
		return &servicediscovery.RegisterInstanceOutput{}, nil
	}
}

func (s *SrvServer) DeregisterInstance(ctx context.Context, params *servicediscovery.DeregisterInstanceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.DeregisterInstanceOutput, error) {
	if srv := s.getServiceById(*params.ServiceId); srv == nil {
		return nil, xerrors.Errorf("service not found: %s", *params.ServiceId)
	} else {
		insts := s.SrvInsts[*srv.Name]
		for i, inst := range insts {
			if *inst.Id == *params.InstanceId {
				insts = append(insts[:i], insts[i+1:]...)
				s.SrvInsts[*params.ServiceId] = insts
				delete(s.SrvInstHelths, *params.InstanceId)
				return &servicediscovery.DeregisterInstanceOutput{}, nil
			}
		}
		return nil, xerrors.Errorf("instance not found: %s", *params.InstanceId)
	}
}

func (s *SrvServer) GetService(ctx context.Context, params *servicediscovery.GetServiceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.GetServiceOutput, error) {
	svc := s.getServiceById(*params.Id)
	if svc == nil {
		return nil, xerrors.Errorf("service not found: %s", *params.Id)
	}
	return &servicediscovery.GetServiceOutput{Service: svc}, nil
}

func (s *SrvServer) GetNamespace(ctx context.Context, params *servicediscovery.GetNamespaceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.GetNamespaceOutput, error) {
	for _, ns := range s.SrvNamespaces {
		if *ns.Id == *params.Id {
			return &servicediscovery.GetNamespaceOutput{Namespace: ns}, nil
		}
	}
	return nil, xerrors.Errorf("namespace not found: %s", *params.Id)
}

package policy

import (
	"reflect"
	"sort"

	"github.com/loilo-inc/canarycage/awsiface"
)

type Statement struct {
	Effect   string   `json:"Effect"`
	Action   []string `json:"Action"`
	Resource string   `json:"Resource"`
}

type Document struct {
	Version   string      `json:"Version"`
	Statement []Statement `json:"Statement"`
}

type serviceSpec struct {
	prefix string
	iface  reflect.Type
}

var serviceSpecs = []serviceSpec{
	{prefix: "ecs", iface: reflect.TypeFor[awsiface.EcsClient]()},
	{prefix: "ecr", iface: reflect.TypeFor[awsiface.EcrClient]()},
	{prefix: "elbv2", iface: reflect.TypeFor[awsiface.AlbClient]()},
	{prefix: "ec2", iface: reflect.TypeFor[awsiface.Ec2Client]()},
}

func DefaultDocument() Document {
	statements := make([]Statement, 0, len(serviceSpecs))
	for _, spec := range serviceSpecs {
		actions := actionsForInterface(spec.prefix, spec.iface)
		statements = append(statements, Statement{
			Effect:   "Allow",
			Action:   actions,
			Resource: "*",
		})
	}
	return Document{
		Version:   "2012-10-17",
		Statement: statements,
	}
}

func actionsForInterface(prefix string, iface reflect.Type) []string {
	actions := make([]string, 0, iface.NumMethod())
	for i := 0; i < iface.NumMethod(); i++ {
		method := iface.Method(i)
		actions = append(actions, prefix+":"+method.Name)
	}
	sort.Strings(actions)
	return actions
}

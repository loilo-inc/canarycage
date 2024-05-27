package test

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

type TaskDefinitionRepository struct {
	families map[string]*TaskDefinitionFamily
}

type TaskDefinitionFamily struct {
	family    string
	revision  int32
	revisions map[int32]*ecstypes.TaskDefinition
}

func (t *TaskDefinitionRepository) Register(input *ecs.RegisterTaskDefinitionInput) (*ecstypes.TaskDefinition, error) {
	family := *input.Family
	if _, ok := t.families[family]; !ok {
		t.families[family] = &TaskDefinitionFamily{
			family:    family,
			revisions: make(map[int32]*ecstypes.TaskDefinition),
		}
	}
	return t.families[family].Register(input)
}

func parseTaskDefinitionArn(arn string) (string, int32) {
	if regexp.MustCompile(`arn:aws:ecs:.*:.*:task-definition/.*:\d+`).MatchString(arn) {
		split := strings.Split(arn, "/")
		familyRev := split[len(split)-1]
		split = strings.Split(familyRev, ":")
		family := split[0]
		revision, _ := strconv.ParseInt(split[1], 10, 32)
		return family, int32(revision)
	} else if regexp.MustCompile(`.*:\d+`).MatchString(arn) {
		split := strings.Split(arn, ":")
		family := split[0]
		revision, _ := strconv.ParseInt(split[1], 10, 32)
		return family, int32(revision)
	}
	return "", 0
}

func (t *TaskDefinitionRepository) Get(familyRev string) *ecstypes.TaskDefinition {
	family, revision := parseTaskDefinitionArn(familyRev)
	if f, ok := t.families[family]; !ok {
		return nil
	} else if td, ok := f.revisions[int32(revision)]; !ok {
		return nil
	} else {
		return td
	}
}

func (t *TaskDefinitionRepository) List() []*ecstypes.TaskDefinition {
	var tds []*ecstypes.TaskDefinition
	for _, f := range t.families {
		for _, td := range f.revisions {
			tds = append(tds, td)
		}
	}
	return tds
}

func (t *TaskDefinitionFamily) Register(input *ecs.RegisterTaskDefinitionInput) (*ecstypes.TaskDefinition, error) {
	t.revision++
	arn := fmt.Sprintf("arn:aws:ecs:us-west-2:012345678910:task-definition/%s:%d", t.family, t.revision)
	td := &ecstypes.TaskDefinition{
		TaskDefinitionArn:    &arn,
		Family:               &t.family,
		Revision:             t.revision,
		ContainerDefinitions: input.ContainerDefinitions,
	}
	t.revisions[t.revision] = td
	return td, nil
}

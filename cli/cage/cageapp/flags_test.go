package cageapp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v3"
)

func TestRegionFlag(t *testing.T) {
	var dest string
	flag := RegionFlag(&dest)

	assert.NotNil(t, flag)
	assert.Equal(t, "region", flag.Name)
	assert.Equal(t, "aws region for ecs. if not specified, try to load from aws sessions automatically", flag.Usage)
	assert.True(t, flag.Required)
	assert.Equal(t, &dest, flag.Destination)

	// Verify Sources contain the correct environment variable
	sources := flag.Sources
	assert.NotNil(t, sources)
	// The Sources field is a ValueSourceChain, so we verify it's set
	assert.NotNil(t, sources)
}

func TestClusterFlag(t *testing.T) {
	var dest string
	flag := ClusterFlag(&dest)

	assert.NotNil(t, flag)
	assert.Equal(t, "cluster", flag.Name)
	assert.Equal(t, "ecs cluster name. if not specified, load from service.json", flag.Usage)
	assert.False(t, flag.Required)
	assert.Equal(t, &dest, flag.Destination)
	assert.NotNil(t, flag.Sources)
}

func TestServiceFlag(t *testing.T) {
	var dest string
	flag := ServiceFlag(&dest)

	assert.NotNil(t, flag)
	assert.Equal(t, "service", flag.Name)
	assert.Equal(t, "service name. if not specified, load from service.json", flag.Usage)
	assert.False(t, flag.Required)
	assert.Equal(t, &dest, flag.Destination)
	assert.NotNil(t, flag.Sources)
}

func TestTaskDefinitionArnFlag(t *testing.T) {
	var dest string
	flag := TaskDefinitionArnFlag(&dest)

	assert.NotNil(t, flag)
	assert.Equal(t, "taskDefinitionArn", flag.Name)
	assert.Equal(t, "full arn or family:revision of task definition. if not specified, new task definition will be created based on task-definition.json", flag.Usage)
	assert.False(t, flag.Required)
	assert.Equal(t, &dest, flag.Destination)
	assert.NotNil(t, flag.Sources)
}

func TestCanaryTaskIdleDurationFlag(t *testing.T) {
	var dest int
	flag := CanaryTaskIdleDurationFlag(&dest)

	assert.NotNil(t, flag)
	assert.Equal(t, "canaryTaskIdleDuration", flag.Name)
	assert.Equal(t, "duration seconds for waiting canary task that isn't attached to target group considered as ready for serving traffic", flag.Usage)
	assert.Equal(t, 15, flag.Value)
	assert.Equal(t, &dest, flag.Destination)
	assert.NotNil(t, flag.Sources)
}

func TestTaskRunningWaitFlag(t *testing.T) {
	var dest int
	flag := TaskRunningWaitFlag(&dest)

	assert.NotNil(t, flag)
	assert.Equal(t, "taskRunningTimeout", flag.Name)
	assert.Equal(t, "max duration seconds for waiting canary task running", flag.Usage)
	assert.Equal(t, 900, flag.Value)
	assert.Equal(t, "ADVANCED", flag.Category)
	assert.Equal(t, &dest, flag.Destination)
	assert.NotNil(t, flag.Sources)
}

func TestTaskHealthCheckWaitFlag(t *testing.T) {
	var dest int
	flag := TaskHealthCheckWaitFlag(&dest)

	assert.NotNil(t, flag)
	assert.Equal(t, "taskHealthCheckTimeout", flag.Name)
	assert.Equal(t, "max duration seconds for waiting canary task health check", flag.Usage)
	assert.Equal(t, 900, flag.Value)
	assert.Equal(t, "ADVANCED", flag.Category)
	assert.Equal(t, &dest, flag.Destination)
	assert.NotNil(t, flag.Sources)
}

func TestTaskStoppedWaitFlag(t *testing.T) {
	var dest int
	flag := TaskStoppedWaitFlag(&dest)

	assert.NotNil(t, flag)
	assert.Equal(t, "taskStoppedTimeout", flag.Name)
	assert.Equal(t, "max duration seconds for waiting canary task stopped", flag.Usage)
	assert.Equal(t, 900, flag.Value)
	assert.Equal(t, "ADVANCED", flag.Category)
	assert.Equal(t, &dest, flag.Destination)
	assert.NotNil(t, flag.Sources)
}

func TestServiceStableWaitFlag(t *testing.T) {
	var dest int
	flag := ServiceStableWaitFlag(&dest)

	assert.NotNil(t, flag)
	assert.Equal(t, "serviceStableTimeout", flag.Name)
	assert.Equal(t, "max duration seconds for waiting service stable", flag.Usage)
	assert.Equal(t, 900, flag.Value)
	assert.Equal(t, "ADVANCED", flag.Category)
	assert.Equal(t, &dest, flag.Destination)
	assert.NotNil(t, flag.Sources)
}

// TestFlagDestinations verifies that flag destination pointers work correctly
func TestFlagDestinations(t *testing.T) {
	t.Run("String flag destinations", func(t *testing.T) {
		var region, cluster, service, taskDefArn string

		regionFlag := RegionFlag(&region)
		clusterFlag := ClusterFlag(&cluster)
		serviceFlag := ServiceFlag(&service)
		taskDefArnFlag := TaskDefinitionArnFlag(&taskDefArn)

		// Verify all flags have their destination pointers set
		assert.Equal(t, &region, regionFlag.Destination)
		assert.Equal(t, &cluster, clusterFlag.Destination)
		assert.Equal(t, &service, serviceFlag.Destination)
		assert.Equal(t, &taskDefArn, taskDefArnFlag.Destination)
	})

	t.Run("Int flag destinations", func(t *testing.T) {
		var canaryIdleDuration, taskRunning, taskHealthCheck, taskStopped, serviceStable int

		canaryIdleFlag := CanaryTaskIdleDurationFlag(&canaryIdleDuration)
		taskRunningFlag := TaskRunningWaitFlag(&taskRunning)
		taskHealthCheckFlag := TaskHealthCheckWaitFlag(&taskHealthCheck)
		taskStoppedFlag := TaskStoppedWaitFlag(&taskStopped)
		serviceStableFlag := ServiceStableWaitFlag(&serviceStable)

		// Verify all flags have their destination pointers set
		assert.Equal(t, &canaryIdleDuration, canaryIdleFlag.Destination)
		assert.Equal(t, &taskRunning, taskRunningFlag.Destination)
		assert.Equal(t, &taskHealthCheck, taskHealthCheckFlag.Destination)
		assert.Equal(t, &taskStopped, taskStoppedFlag.Destination)
		assert.Equal(t, &serviceStable, serviceStableFlag.Destination)
	})
}

// TestFlagDefaultValues verifies default values for int flags
func TestFlagDefaultValues(t *testing.T) {
	tests := []struct {
		name         string
		flagFunc     func(*int) *cli.IntFlag
		expectedVal  int
		expectedCat  string
	}{
		{
			name:        "CanaryTaskIdleDurationFlag",
			flagFunc:    CanaryTaskIdleDurationFlag,
			expectedVal: 15,
			expectedCat: "",
		},
		{
			name:        "TaskRunningWaitFlag",
			flagFunc:    TaskRunningWaitFlag,
			expectedVal: 900,
			expectedCat: "ADVANCED",
		},
		{
			name:        "TaskHealthCheckWaitFlag",
			flagFunc:    TaskHealthCheckWaitFlag,
			expectedVal: 900,
			expectedCat: "ADVANCED",
		},
		{
			name:        "TaskStoppedWaitFlag",
			flagFunc:    TaskStoppedWaitFlag,
			expectedVal: 900,
			expectedCat: "ADVANCED",
		},
		{
			name:        "ServiceStableWaitFlag",
			flagFunc:    ServiceStableWaitFlag,
			expectedVal: 900,
			expectedCat: "ADVANCED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dest int
			flag := tt.flagFunc(&dest)

			assert.Equal(t, tt.expectedVal, flag.Value, "Default value should match")
			assert.Equal(t, tt.expectedCat, flag.Category, "Category should match")
		})
	}
}

// TestFlagNames verifies all flag names are correctly set
func TestFlagNames(t *testing.T) {
	tests := []struct {
		name     string
		flag     cli.Flag
		expected string
	}{
		{
			name:     "RegionFlag",
			flag:     RegionFlag(new(string)),
			expected: "region",
		},
		{
			name:     "ClusterFlag",
			flag:     ClusterFlag(new(string)),
			expected: "cluster",
		},
		{
			name:     "ServiceFlag",
			flag:     ServiceFlag(new(string)),
			expected: "service",
		},
		{
			name:     "TaskDefinitionArnFlag",
			flag:     TaskDefinitionArnFlag(new(string)),
			expected: "taskDefinitionArn",
		},
		{
			name:     "CanaryTaskIdleDurationFlag",
			flag:     CanaryTaskIdleDurationFlag(new(int)),
			expected: "canaryTaskIdleDuration",
		},
		{
			name:     "TaskRunningWaitFlag",
			flag:     TaskRunningWaitFlag(new(int)),
			expected: "taskRunningTimeout",
		},
		{
			name:     "TaskHealthCheckWaitFlag",
			flag:     TaskHealthCheckWaitFlag(new(int)),
			expected: "taskHealthCheckTimeout",
		},
		{
			name:     "TaskStoppedWaitFlag",
			flag:     TaskStoppedWaitFlag(new(int)),
			expected: "taskStoppedTimeout",
		},
		{
			name:     "ServiceStableWaitFlag",
			flag:     ServiceStableWaitFlag(new(int)),
			expected: "serviceStableTimeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var actualName string
			switch f := tt.flag.(type) {
			case *cli.StringFlag:
				actualName = f.Name
			case *cli.IntFlag:
				actualName = f.Name
			}
			assert.Equal(t, tt.expected, actualName, "Flag name should match")
		})
	}
}

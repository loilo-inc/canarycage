package cage_test

// func TestCage_CreateNextTaskDefinition(t *testing.T) {
// 	envars := &cage.Envars{
// 		TaskDefinitionArn: "arn://task",
// 	}
// 	ctrl := gomock.NewController(t)
// 	e := mock_awsiface.NewMockEcsClient(ctrl)
// 	e.EXPECT().DescribeTaskDefinition(gomock.Any(), gomock.Any()).Return(
// 		&ecs.DescribeTaskDefinitionOutput{
// 			TaskDefinition: &ecstypes.TaskDefinition{TaskDefinitionArn: aws.String("arn://task")},
// 		}, nil)
// 	// nextTaskDefinitionArnがある場合はdescribeTaskDefinitionから返す
// 	cagecli := cage.NewCage(&cage.Input{
// 		Env: envars,
// 		ECS: e,
// 	})
// 	o, err := cagecli.CreateNextTaskDefinition(context.Background())
// 	if err != nil {
// 		t.Fatalf(err.Error())
// 	}
// 	assert.Equal(t, envars.TaskDefinitionArn, *o.TaskDefinitionArn)
// }

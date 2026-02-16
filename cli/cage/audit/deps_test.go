package audit

import (
	"context"
	"testing"

	"github.com/loilo-inc/canarycage/v5/cli/cage/cageapp"
)

func TestProvideAuditCmd(t *testing.T) {
	ctx := context.Background()
	input := cageapp.NewAuditCmdInput()
	input.Region = "us-east-1"
	audit, err := ProvideAuditCmd(ctx, input)
	if err != nil {
		t.Fatalf("ProvideAuditCmd() error = %v, want nil", err)
	}

	if audit == nil {
		t.Fatal("ProvideAuditCmd() returned nil audit")
	}
}

func TestProvideAuditCmd_WithDifferentRegions(t *testing.T) {
	regions := []string{"us-east-1", "eu-west-1", "ap-northeast-1"}

	for _, region := range regions {
		t.Run(region, func(t *testing.T) {
			ctx := context.Background()
			input := cageapp.NewAuditCmdInput()
			input.Region = region

			audit, err := ProvideAuditCmd(ctx, input)
			if err != nil {
				t.Fatalf("ProvideAuditCmd() error = %v, want nil", err)
			}

			if audit == nil {
				t.Fatalf("ProvideAuditCmd() returned nil audit for region %s", region)
			}
		})
	}
}

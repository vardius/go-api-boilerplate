package executioncontext

import (
	"context"
	"testing"
)

func TestWithFlag(t *testing.T) {
	ctx := context.Background()
	ctx = WithFlag(ctx, LIVE)

	if !Has(ctx, LIVE) {
		t.Fail()
	}
}

func TestHas(t *testing.T) {
	ctx := context.Background()
	if Has(ctx, LIVE) {
		t.Fail()
	}

	ctx = WithFlag(ctx, LIVE)
	ctx = WithFlag(ctx, REPLAY)

	if !Has(ctx, LIVE) {
		t.Fail()
	}
}

func TestFromContext(t *testing.T) {
	ctx := context.Background()
	if FromContext(ctx) != 0 {
		t.Fail()
	}

	ctx = WithFlag(ctx, LIVE)
	ctx = WithFlag(ctx, REPLAY)

	if FromContext(ctx)&(LIVE|REPLAY) == 0 {
		t.Fail()
	}
}

func TestToggleFlag(t *testing.T) {
	type args struct {
		ctx  context.Context
		flag Flag
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"on", args{context.Background(), LIVE}, true},
		{"off", args{WithFlag(context.Background(), LIVE), LIVE}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if ctx := ToggleFlag(tt.args.ctx, tt.args.flag); Has(ctx, tt.args.flag) != tt.want {
				t.Errorf("ToggleFlag() = (%d) %v, got %v, want %v", FromContext(ctx), ctx, Has(ctx, tt.args.flag), tt.want)
			}
		})
	}
}

func TestClearFlag(t *testing.T) {
	type args struct {
		ctx  context.Context
		flag Flag
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"on", args{context.Background(), LIVE}, true},
		{"off", args{WithFlag(context.Background(), LIVE), LIVE}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if ctx := ClearFlag(tt.args.ctx, tt.args.flag); Has(ctx, tt.args.flag) {
				t.Errorf("ToggleFlag() = (%d) %v, got %v, want %v", FromContext(ctx), ctx, Has(ctx, tt.args.flag), tt.want)
			}
		})
	}
}

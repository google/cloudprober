package runconfig

import (
	"testing"

	"google.golang.org/grpc"
)

func TestRunConfig(t *testing.T) {
	Init()
	if srv := DefaultGRPCServer(); srv != nil {
		t.Fatalf("RunConfig has server unexpectedly set. Got %v Want nil", srv)
	}
	testSrv := grpc.NewServer()
	if testSrv == nil {
		t.Fatal("Unable to create a test gRPC server")
	}
	if err := SetDefaultGRPCServer(testSrv); err != nil {
		t.Fatalf("Unable to set default gRPC server: %v", err)
	}
	if srv := DefaultGRPCServer(); srv != testSrv {
		t.Fatalf("Error retrieving stored service. Got %v Want %v", srv, testSrv)
	}
	if err := SetDefaultGRPCServer(testSrv); err == nil {
		t.Errorf("RunConfig allowed overriding of an already set variable.")
	}
}

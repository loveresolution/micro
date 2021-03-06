// +build integration

package test

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func TestStore(t *testing.T) {
	TrySuite(t, testStore, 5)
}

func testStore(t *T) {
	t.Parallel()
	serv := NewServer(t, WithLogin())
	defer serv.Close()
	if err := serv.Run(); err != nil {
		return
	}

	cmd := serv.Command()

	// Execute first command in read to wait for store service
	// to start up
	if err := Try("Calling micro store read", t, func() ([]byte, error) {
		outp, err := cmd.Exec("store", "read", "somekey")
		if err == nil {
			return outp, errors.New("store read should fail")
		}
		if !strings.Contains(string(outp), "not found") {
			return outp, fmt.Errorf("Output should be 'not found', got %v", string(outp))
		}
		return outp, nil
	}, 8*time.Second); err != nil {
		return
	}

	outp, err := cmd.Exec("store", "write", "somekey", "val1")
	if err != nil {
		t.Fatal(string(outp))
		return
	}
	if string(outp) != "" {
		t.Fatalf("Expected no output, got: %v", string(outp))
		return
	}

	outp, err = cmd.Exec("store", "read", "somekey")
	if err != nil {
		t.Fatal(string(outp))
		return
	}
	if string(outp) != "val1\n" {
		t.Fatalf("Expected 'val1\n', got: '%v'", string(outp))
		return
	}

	outp, err = cmd.Exec("store", "delete", "somekey")
	if err != nil {
		t.Fatal(err)
		return
	}
	if string(outp) != "" {
		t.Fatalf("Expected '', got: '%v'", string(outp))
		return
	}

	outp, err = cmd.Exec("store", "read", "somekey")
	if err == nil {
		t.Fatalf("store read should fail: %v", string(outp))
		return
	}
	if !strings.Contains(string(outp), "not found") {
		t.Fatalf("Expected 'not found\n', got: '%v'", string(outp))
		return
	}

	// Test prefixes
	outp, err = cmd.Exec("store", "write", "somekey1", "val1")
	if err != nil {
		t.Fatal(string(outp))
		return
	}
	if string(outp) != "" {
		t.Fatalf("Expected no output, got: %v", string(outp))
		return
	}

	outp, err = cmd.Exec("store", "write", "somekey2", "val2")
	if err != nil {
		t.Fatal(string(outp))
		return
	}
	if string(outp) != "" {
		t.Fatalf("Expected no output, got: %v", string(outp))
		return
	}

	// Read exact key
	outp, err = cmd.Exec("store", "read", "somekey")
	if err == nil {
		t.Fatalf("store read should fail: %v", string(outp))
		return
	}
	if !strings.Contains(string(outp), "not found") {
		t.Fatalf("Expected 'not found\n', got: '%v'", string(outp))
		return
	}

	outp, err = cmd.Exec("store", "read", "--prefix", "somekey")
	if err != nil {
		t.Fatalf("store prefix read not should fail: %v", string(outp))
		return
	}
	if string(outp) != "val1\nval2\n" {
		t.Fatalf("Expected output not present, got: '%v'", string(outp))
		return
	}

	outp, err = cmd.Exec("store", "read", "-v", "--prefix", "somekey")
	if err != nil {
		t.Fatalf("store prefix read not should fail: %v", string(outp))
		return
	}
	if !strings.Contains(string(outp), "somekey1") || !strings.Contains(string(outp), "somekey2") ||
		!strings.Contains(string(outp), "val1") || !strings.Contains(string(outp), "val2") {
		t.Fatalf("Expected output not present, got: '%v'", string(outp))
		return
	}

	outp, err = cmd.Exec("store", "list")
	if err != nil {
		t.Fatalf("store list should not fail: %v", string(outp))
		return
	}
	if !strings.Contains(string(outp), "somekey1") || !strings.Contains(string(outp), "somekey2") {
		t.Fatalf("Expected output not present, got: '%v'", string(outp))
		return
	}

}

func TestStoreImpl(t *testing.T) {
	TrySuite(t, testStoreImpl, 5)
}

func testStoreImpl(t *T) {
	t.Parallel()
	serv := NewServer(t, WithLogin())
	defer serv.Close()
	if err := serv.Run(); err != nil {
		return
	}

	cmd := serv.Command()

	runTarget := "./service/storeexample"
	branch := "latest"
	if os.Getenv("MICRO_IS_KIND_TEST") == "true" {
		if ref := os.Getenv("GITHUB_REF"); len(ref) > 0 {
			branch = strings.TrimPrefix(ref, "refs/heads/")
		} else {
			branch = "master"
		}
		runTarget = "github.com/micro/micro/test/service/storeexample@" + branch
		t.Logf("Running service from the %v branch of micro", branch)
	}

	outp, err := cmd.Exec("run", runTarget)
	if err != nil {
		t.Fatalf("micro run failure, output: %v", string(outp))
		return
	}

	if err := Try("Find storeexample", t, func() ([]byte, error) {
		outp, err := cmd.Exec("status")
		if err != nil {
			return outp, err
		}

		// The started service should have the runtime name of "service/example",
		// as the runtime name is the relative path inside a repo.
		if !statusRunning("storeexample", branch, outp) {
			return outp, errors.New("Can't find example service in runtime")
		}
		return outp, err
	}, 15*time.Second); err != nil {
		return
	}

	if err := Try("Check logs", t, func() ([]byte, error) {
		outp, err := cmd.Exec("logs", "storeexample")
		if err != nil {
			return nil, err
		}
		if !strings.Contains(string(outp), "Listening on") {
			return nil, fmt.Errorf("Service not ready")
		}
		return nil, nil
	}, 60*time.Second); err != nil {
		return
	}
	outp, err = cmd.Exec("call", "--request_timeout=15s", "example", "Example.TestExpiry")
	if err != nil {
		t.Fatalf("Error %s, %s", err, outp)
	}

	outp, err = cmd.Exec("call", "--request_timeout=15s", "example", "Example.TestList")
	if err != nil {
		t.Fatalf("Error %s, %s", err, outp)
	}

	outp, err = cmd.Exec("call", "--request_timeout=15s", "example", "Example.TestListLimit")
	if err != nil {
		t.Fatalf("Error %s, %s", err, outp)
	}
	outp, err = cmd.Exec("call", "--request_timeout=15s", "example", "Example.TestListOffset")
	if err != nil {
		t.Fatalf("Error %s, %s", err, outp)
	}

}

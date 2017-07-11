package netns

import (
	"testing"

	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"time"
)

func TestCreateNetworkNamespaceTarget(t *testing.T) {
	target := fmt.Sprintf("/tmp/%d", rand.New(rand.NewSource(time.Now().UnixNano())).Int())
	defer os.Remove(target)
	err := createNetworkNamespaceTarget(target)
	if err != nil {
		t.Errorf("%s", err)
	}
	_, err = os.Stat(target)
	if err != nil {
		t.Errorf("%s", err)
	}
}

func TestGetCurrentNetworkNamespace(t *testing.T) {
	ns, ref, err := getCurrentNetworkNamespace()
	if err != nil {
		t.Errorf("%s", err)
	}
	_, err = os.Stat(ns)
	if err != nil {
		t.Errorf("%s", err)
	}
	_, err = os.Stat(ns)
	if err != nil {
		t.Errorf("%s", err)
	}
	if ref == "" {
		t.Errorf("ref == %s", ref)
	}
}

func TestCreateNetworkNamespaceInFork(t *testing.T) {
	err := InitNetworkNamespaceDirectory()
	if err != nil {
		t.Errorf("%s", err)
	}
	ns, ref, err := getCurrentNetworkNamespace()
	if err != nil {
		t.Errorf("%s", err)
	}
	target := fmt.Sprintf("test_%s", strconv.Itoa(rand.New(rand.NewSource(time.Now().UnixNano())).Int()))
	err = CreateNetworkNamespaceInFork(target)
	defer exec.Command("ip", "netns", "del", target).Run()
	if err != nil {
		t.Errorf("%s", err)
	}
	newNs, newRef, err := getCurrentNetworkNamespace()
	if err != nil {
		t.Errorf("%s", err)
	}
	if newNs != ns {
		t.Errorf("%s != %s", newNs, ns)
	}
	if newRef != ref {
		t.Errorf("%s != %s", newRef, ref)
	}
}

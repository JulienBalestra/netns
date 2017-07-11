package netns

//#include <unistd.h>
import "C"

import (
	"fmt"
	"github.com/golang/glog"
	"golang.org/x/sys/unix"
	"os"
	"path"
	"runtime"
	"syscall"
)

const networkNamespaceDirectory = "/var/run/netns"

// If missing, creates a directory to store netns targets
func InitNetworkNamespaceDirectory() error {
	err := os.MkdirAll(networkNamespaceDirectory, 0755)
	if err != nil {
		glog.Errorf("fail to create networkNamespaceDirectory %q: %s", networkNamespaceDirectory, err)
		return err
	}
	glog.V(4).Infof("created networkNamespaceDirectory %q", networkNamespaceDirectory)
	return err
}

// Create an empty regular file as network namespace target
func createNetworkNamespaceTarget(path string) error {
	_, err := os.Stat(path)
	if !os.IsNotExist(err) {
		return fmt.Errorf("path %s already exist", path)
	}

	namespaceTarget, err := os.Create(path)
	defer namespaceTarget.Close()
	if err != nil {
		return err
	}
	return nil
}

// Get the current network namespace absolute FS path
func getCurrentNetworkNamespace() (netnsPath string, netnsRef string, err error) {
	netnsPath = fmt.Sprintf("/proc/%d/task/%d/ns/net", os.Getpid(), unix.Gettid())
	netnsRef, err = os.Readlink(netnsPath)
	if err != nil {
		glog.Errorf("fail to get current networkNamespace %q: %s", netnsPath, err)
		return netnsPath, netnsRef, err
	}
	return netnsPath, netnsRef, nil
}

// Create a new network namespace and mount it to the target in argument
func switchNetworkNamespace(target string) error {
	var newNetNsPath, newNetNsRef, originNetNsPath, originNetNsRef string
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	originNetNsPath, originNetNsRef, err := getCurrentNetworkNamespace()
	if err != nil {
		return err
	}
	glog.V(4).Infof("origin netns: %q -> %q", originNetNsPath, originNetNsRef)

	err = unix.Unshare(unix.CLONE_NEWNET)
	if err != nil {
		glog.Errorf("fail unix.CLONE_NEWNET for %s: %s", target, err)
		return err
	}

	newNetNsPath, newNetNsRef, err = getCurrentNetworkNamespace()
	if err != nil {
		return err
	}
	glog.V(4).Infof("new netns: %q -> %q", newNetNsPath, newNetNsRef)

	err = unix.Mount(newNetNsPath, target, "none", unix.MS_BIND, "")
	if err != nil {
		glog.Errorf("fail to mount %s to %s: %s", newNetNsPath, target, err)
		err := unix.Unmount(target, unix.MNT_DETACH)
		if err != nil {
			glog.Errorf("fail to unmount %s during error handling raise %s", target, err)
		}
		return err
	}
	return nil
}

// Create a new network namespace with the name passed in parameter
func CreateNetworkNamespace(name string) (err error) {
	nsTarget := path.Join(networkNamespaceDirectory, name)
	err = createNetworkNamespaceTarget(nsTarget)
	if err != nil {
		glog.Errorf("fail to create networkNamespaceTarget %q: %s", nsTarget, err)
		return err
	}
	glog.V(4).Infof("created networkNamespaceTarget %q", nsTarget)
	err = switchNetworkNamespace(nsTarget)
	if err != nil {
		errRemove := os.Remove(nsTarget)
		if errRemove != nil {
			glog.Errorf("fail to remove networkNamespaceTarget %s: %s", nsTarget, errRemove)
		}
	}
	return err
}

// Create a new network namespace in a dedicated process without switching on it
func CreateNetworkNamespaceInFork(name string) (err error) {
	pid := C.fork()
	if pid == 0 {
		err = CreateNetworkNamespace(name)
		if err != nil {
			glog.Errorf("exiting on error during namespace creation: %q", err)
			os.Exit(3)
		}
		os.Exit(0)
	}
	var wait syscall.WaitStatus
	wait = 0
	syscall.Wait4(int(pid), &wait, 0, nil)
	if wait.ExitStatus() != 0 {
		return fmt.Errorf("fail to create NetworkNamespace")
	}
	return nil
}

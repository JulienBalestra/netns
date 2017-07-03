package netns

import (
	"fmt"
	"github.com/golang/glog"
	"golang.org/x/sys/unix"
	"os"
	"path"
	"runtime"
)

const (
	networkNamespaceDirectory = "/var/run/netns"

	switchErrNil = iota
	switchErrCloneNewNamespace
	switchErrGetOriginNamespace
	switchErrGetNewNamespace
	switchErrMountNewNamespace
)

type switchError struct {
	err   error
	errNo int
}

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
func getCurrentNetworkNamespace() (netnsPath string, netnsInode string, err error) {
	netnsPath = fmt.Sprintf("/proc/%d/task/%d/ns/net", os.Getpid(), unix.Gettid())
	netnsInode, err = os.Readlink(netnsPath)
	if err != nil {
		glog.Errorf("fail to get current networkNamespace %q: %s", netnsPath, err)
		return netnsPath, netnsInode, err
	}
	return netnsPath, netnsInode, nil
}

// This function needs to run as Routine to creates a new network namespace without continue to use it
// Errors are sent to channel
func switchNetworkNamespace(target string, errChan chan<- switchError) {
	var switchErr switchError
	var newNetNsPath, newNetNsInode, originNetNsPath, originNetNsInode string

	// Avoid changing thread
	runtime.LockOSThread()

	originNetNsPath, originNetNsInode, switchErr.err = getCurrentNetworkNamespace()
	if switchErr.err != nil {
		switchErr.errNo = switchErrGetOriginNamespace
		errChan <- switchErr
		return
	}
	glog.V(4).Infof("origin OSThread netns: %q -> %q", originNetNsPath, originNetNsInode)

	switchErr.err = unix.Unshare(unix.CLONE_NEWNET)
	if switchErr.err != nil {
		glog.Errorf("fail unix.CLONE_NEWNET for %s: %s", target, switchErr.err)
		switchErr.errNo = switchErrCloneNewNamespace
		errChan <- switchErr
		return
	}

	newNetNsPath, newNetNsInode, switchErr.err = getCurrentNetworkNamespace()
	if switchErr.err != nil {
		switchErr.errNo = switchErrGetNewNamespace
		errChan <- switchErr
		return
	}
	glog.V(4).Infof("new OSThread netns: %q -> %q", newNetNsPath, newNetNsInode)

	switchErr.err = unix.Mount(newNetNsPath, target, "none", unix.MS_BIND, "")
	if switchErr.err != nil {
		glog.Errorf("fail to mount %s to %s: %s", newNetNsPath, target, switchErr.err)
		switchErr.errNo = switchErrMountNewNamespace
		errChan <- switchErr
	}
	switchErr.err = nil
	switchErr.errNo = switchErrNil
	errChan <- switchErr
}

func cleanNetworkNamespaceRessources(target string, errSwitch switchError) {
	if errSwitch.errNo == switchErrMountNewNamespace {
		err := unix.Unmount(target, unix.MNT_DETACH)
		if err != nil {
			glog.Errorf("fail to unmount %s during error handling raise %s", target, err)
		}
	}
	err := os.Remove(target)
	if err != nil {
		glog.Errorf("fail to remove networkNamespaceTarget %s: %s", target, err)
	}
}

func CreateNetworkNamespace(name string) (err error) {
	nsTarget := path.Join(networkNamespaceDirectory, name)
	err = createNetworkNamespaceTarget(nsTarget)
	if err != nil {
		glog.Errorf("fail to create networkNamespaceTarget %q: %s", nsTarget, err)
		return err
	}
	glog.V(4).Infof("created networkNamespaceTarget %q", nsTarget)

	errChan := make(chan switchError, 1)
	defer close(errChan)

	go switchNetworkNamespace(nsTarget, errChan)
	errSwitch := <-errChan
	if errSwitch.errNo != switchErrNil {
		cleanNetworkNamespaceRessources(nsTarget, errSwitch)
	}

	return errSwitch.err
}

package netns

import (
	"fmt"
	"github.com/golang/glog"
	"golang.org/x/sys/unix"
	"os"
	"path"
	"runtime"
)

const networkNamespaceDirectory = "/var/run/netns"

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

// create a new network namespace and mount it to the target in argument
func switchNetworkNamespace(target string) error {
	var newNetNsPath, newNetNsInode, originNetNsPath, originNetNsInode string
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	originNetNsPath, originNetNsInode, err := getCurrentNetworkNamespace()
	if err != nil {
		return err
	}
	glog.V(4).Infof("origin netns: %q -> %q", originNetNsPath, originNetNsInode)

	err = unix.Unshare(unix.CLONE_NEWNET)
	if err != nil {
		glog.Errorf("fail unix.CLONE_NEWNET for %s: %s", target, err)
		return err
	}

	newNetNsPath, newNetNsInode, err = getCurrentNetworkNamespace()
	if err != nil {
		return err
	}
	glog.V(4).Infof("new netns: %q -> %q", newNetNsPath, newNetNsInode)

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

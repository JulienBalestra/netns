package main

import (
	"flag"
	"github.com/JulienBalestra/netns/netns"
	"github.com/golang/glog"
	"os"
	"os/exec"
)

func isNetnsAdd(args []string) bool {
	if len(args) < 3 {
		return false
	}
	if args[1] == "netns" && args[2] == "add" {
		return true
	}
	return false
}

func main() {
	if isNetnsAdd(os.Args) == false {
		if len(os.Args) == 1 {
			os.Args = append(os.Args, "help")
		}

		execCommand := exec.Command("ip", os.Args[1:]...)
		output, err := execCommand.CombinedOutput()
		execCommand.Run()
		os.Stdout.Write(output)
		if err != nil {
			os.Exit(2)
		}
		return
	}

	flag.Parse()
	flag.Lookup("alsologtostderr").Value.Set("true")

	err := netns.InitNetworkNamespaceDirectory()
	if err != nil {
		glog.Errorf("exiting on error during init: %q", err)
		os.Exit(2)
	}
	err = netns.CreateNetworkNamespace(os.Args[3])
	if err != nil {
		glog.Errorf("exiting on error during namespace creation: %q", err)
		os.Exit(3)
	}
	return
}

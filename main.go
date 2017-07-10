package main

import (
	"flag"
	"github.com/JulienBalestra/netns/netns"
	"github.com/golang/glog"
	"os"
)

func main() {
	var name = flag.String("name", "", "netns name")
	flag.Parse()
	flag.Lookup("alsologtostderr").Value.Set("true")

	if *name == "" {
		glog.Errorf("provide a netns -name")
		os.Exit(1)
	}

	err := netns.InitNetworkNamespaceDirectory()
	if err != nil {
		glog.Errorf("exiting on error during init: %q", err)
		os.Exit(2)
	}
	err = netns.CreateNetworkNamespaceInFork(*name)
	if err != nil {
		os.Exit(3)
	}
	return
}

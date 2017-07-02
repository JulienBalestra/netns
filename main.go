package main

import (
	"flag"
	"github.com/JulienBalestra/netns/netns"
	"github.com/golang/glog"
)

func main() {
	var name = flag.String("name", "", "netns name")
	flag.Parse()
	flag.Lookup("alsologtostderr").Value.Set("true")

	if *name == "" {
		glog.Errorf("provide a netns -name")
		return
	}

	err := netns.InitNetworkNamespaceDirectory()
	if err != nil {
		glog.Errorf("exiting on error during init: %q", err)
	}
	err = netns.CreateNetworkNamespace(*name)
	if err != nil {
		glog.Errorf("exiting on error during namespace creation: %q", err)
	}
	return
}

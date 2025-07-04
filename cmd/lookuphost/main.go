package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"slices"
)

func main() {
	var fqdn string
	flag.StringVar(&fqdn, "fqdn", "", "Fully Qualified Domain Name to look up")
	flag.Parse()
	fmt.Printf("FQDN: %s\n", fqdn)
	hosts, err := net.LookupHost(fqdn)
	if err != nil {
		log.Fatal(err)
	}
	slices.Sort(hosts)
	for _, host := range hosts {
		println(host)
	}
}

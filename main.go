package main

import (
	"flag"
	"fmt"
	"os"

	"tinygo.org/x/bluetooth"
)

var adapter = bluetooth.DefaultAdapter

var doScan = flag.Bool("scan", false, "Scan for Bluetooth devices")
var addr = flag.String("address", "", "Bluetooth address to connect to")

func main() {
	flag.Parse()

	err := adapter.Enable()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if *doScan {
		scan()
	}

	if *addr != "" {
		connect(*addr)
	}
}

func scan() {
	adapter.Scan(func(a *bluetooth.Adapter, d bluetooth.ScanResult) {
		fmt.Println("device: %s %s", d.Address.String(), d.LocalName())
	})
}

func connect(addr string) {
	var bleResult bluetooth.ScanResult

	adapter.Scan(func(a *bluetooth.Adapter, d bluetooth.ScanResult) {
		fmt.Println("device: %s %s", d.Address.String(), d.LocalName())
		if d.Address.String() == addr {
			adapter.StopScan()
			bleResult = d
		}
	})

	adapter.SetConnectHandler(func(device bluetooth.Addresser, connected bool) {
		fmt.Println("connected", connected)

		return
	})

	_, err := adapter.Connect(bleResult.Address, bluetooth.ConnectionParams{})
	if err != nil {
		panic(err)
	}
}

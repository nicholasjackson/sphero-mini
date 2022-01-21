package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/hashicorp/go-hclog"
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
	log := hclog.New(&hclog.LoggerOptions{Level: hclog.Trace})

	var bleResult bluetooth.ScanResult

	fmt.Println("Discoving device...")
	adapter.Scan(func(a *bluetooth.Adapter, d bluetooth.ScanResult) {
		log.Debug("Found device", "name", d.LocalName(), "address", d.Address.String())
		if d.Address.String() == addr {
			adapter.StopScan()
			bleResult = d
		}
	})

	connected := make(chan bool)

	fmt.Println("Connecting...")
	var device *bluetooth.Device
	var err error
	device, err = adapter.Connect(bleResult.Address, bluetooth.ConnectionParams{})
	if err != nil {
		panic(err)
	}

	services, err := device.DiscoverServices([]bluetooth.UUID{})
	if err != nil {
		panic(err)
	}

	charAPIV2 := getCharacteristic(services, "00010002-574f-4f20-5370-6865726f2121")
	charAntiDOS := getCharacteristic(services, "00020005-574f-4f20-5370-6865726f2121")
	charDFU := getCharacteristic(services, "00020002-574f-4f20-5370-6865726f2121")
	charDFU2 := getCharacteristic(services, "00020004-574f-4f20-5370-6865726f2121")

	sphero := &Sphero{
		charAPIV2:   charAPIV2,
		charAntiDOS: charAntiDOS,
		charDFU:     charDFU,
		charDFU2:    charDFU2,
		log:         log,
	}

	sphero.Wake()

	// wait for connection
	select {
	case <-connected:
		fmt.Println("done")
	}
}

func getCharacteristic(s []bluetooth.DeviceService, uuid string) bluetooth.DeviceCharacteristic {
	uu, err := bluetooth.ParseUUID(uuid)
	if err != nil {
		panic(err)
	}

	for _, s := range s {
		c, err := s.DiscoverCharacteristics([]bluetooth.UUID{uu})
		if err == nil {
			return c[0]
		}
	}

	panic(fmt.Errorf("characteristic: %s not found", uuid))
}

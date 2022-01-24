package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/sphero-mini"
)

var doScan = flag.Bool("scan", false, "Scan for Bluetooth devices")
var addr = flag.String("address", "", "Bluetooth address to connect to")

func main() {
	flag.Parse()

	if *doScan {
		scan()
	}

	if *addr != "" {
		connect(*addr)
	}
}

func scan() {
	ad, err := sphero.NewBluetoothAdapter(createLogger())
	if err != nil {
		fmt.Printf("Unable to create a bluetooth adapter: %s\n", err)
		os.Exit(1)
	}

	sr := ad.Scan()

	for r := range sr {
		fmt.Printf("Found device: %s, address: %s\n", r.Name, r.Address.String())
	}
}

func connect(addr string) {
	logger := createLogger()

	adapter, err := sphero.NewBluetoothAdapter(logger)
	if err != nil {
		fmt.Printf("Unable to create a bluetooth adapter: %s\n", err)
		os.Exit(1)
	}

	s, err := sphero.NewSphero(addr, adapter, logger)
	if err != nil {
		fmt.Printf("Unable to create a new sphero: %s\n", err)
		os.Exit(1)
	}

	s.
		SetLEDColor(235, 64, 52).
		Wait(1*time.Second).
		SetLEDColor(52, 235, 88).
		Wait(1*time.Second).
		SetLEDColor(52, 122, 235).
		Wait(1*time.Second).
		SetLEDColor(0, 0, 0)

	s.Sleep()

	//sphero := &Sphero{
	//	charAPIV2:   charAPIV2,
	//	charAntiDOS: charAntiDOS,
	//	charDFU:     charDFU,
	//	charDFU2:    charDFU2,
	//	log:         log,
	//}

	//sphero.Setup()
	////sphero.GetBatteryVoltage()
	//sphero.SetLEDColor(255, 255, 255)
	//time.Sleep(1 * time.Second)

	//sphero.SetLEDColor(0, 0, 0)
	//time.Sleep(1 * time.Second)

	//sphero.SetLEDColor(255, 255, 255)
	//time.Sleep(1 * time.Second)

	//sphero.SetLEDColor(0, 0, 0)
	//time.Sleep(1 * time.Second)

	//time.Sleep(5 * time.Second)
	//sphero.Sleep()

	//// wait for connection
	//select {
	//case <-connected:
	//	fmt.Println("done")
	//}
}

func createLogger() hclog.Logger {
	return hclog.New(&hclog.LoggerOptions{Level: hclog.Trace, Color: hclog.AutoColor})
}

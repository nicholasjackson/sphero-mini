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

	ball, err := sphero.NewSphero(addr, adapter, logger)
	if err != nil {
		fmt.Printf("Unable to create a new sphero: %s\n", err)
		os.Exit(1)
	}

	// enable the backlight, this is useful to see which direction the sphero is headed
	ball.EnableBackLight()
	time.Sleep(5 * time.Second)

	ball.
		SetLEDColor(235, 64, 52).
		For(1*time.Second).
		SetLEDColor(52, 235, 88).
		For(1*time.Second).
		SetLEDColor(52, 122, 235).
		For(1 * time.Second)

	time.Sleep(5 * time.Second)

	ball.Roll(0, 150).
		For(1*time.Second).
		Roll(180, 150).
		For(1 * time.Second)

	ball.Sleep()
}

func createLogger() hclog.Logger {
	return hclog.New(&hclog.LoggerOptions{Level: hclog.Trace, Color: hclog.AutoColor})
}

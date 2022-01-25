package sphero

import (
	"fmt"
	"time"

	"github.com/hashicorp/go-hclog"
	"tinygo.org/x/bluetooth"
)

var defaultAdapter = bluetooth.DefaultAdapter
var connectionTimeout = 60 * time.Second

// BluetoothAdapter allows the interaction with the phyiscal bluetooth stack
type BluetoothAdapter struct {
	adapter     *bluetooth.Adapter
	log         hclog.Logger
	scanResult  chan ScanResult
	scanStopped chan bool
}

// ScanResult is returned from the Scan function and encapsulates the
// details of a Bluetooth device
type ScanResult struct {
	Name    string
	Address bluetooth.Addresser
}

// NewBluetoothAdapter creates and initializes the default bluetooth adapter on the machine
func NewBluetoothAdapter(l hclog.Logger) (*BluetoothAdapter, error) {
	err := defaultAdapter.Enable()
	if err != nil {
		l.Error("Unable to enable the bluetooth adapter", "error", err)
		return nil, err
	}

	return &BluetoothAdapter{adapter: defaultAdapter, log: l, scanResult: make(chan ScanResult)}, nil
}

// Scan for bluetooth devices, this method returns a channel of ScanResult
// that can be constantly itterated over to print the devices
func (b *BluetoothAdapter) Scan() chan ScanResult {
	b.scanStopped = make(chan bool, 1)

	go func(results chan ScanResult, stop chan bool) {
		err := b.adapter.Scan(func(a *bluetooth.Adapter, d bluetooth.ScanResult) {
			name := d.LocalName()
			if name == "" {
				name = "UNKNOWN"
			}

			select {
			case <-stop:
				return
			default:
			}

			results <- ScanResult{Name: name, Address: d.Address}
		})

		if err != nil {
			panic(err)
		}
	}(b.scanResult, b.scanStopped)

	return b.scanResult
}

// StopScanning stops the scanning process and closes the ScanResult channel
// returned by the Scan function
func (b *BluetoothAdapter) StopScanning() {
	b.adapter.StopScan()
	b.scanStopped <- true
}

// Connect to a bluetooth device
func (b *BluetoothAdapter) Connect(addr bluetooth.Addresser) (*bluetooth.Device, error) {
	b.adapter.SetConnectHandler(func(device bluetooth.Addresser, connected bool) {
		b.log.Trace("Connection status changed", "connected", connected)
	})

	// TODO, connection timeout on Darwin does not seem to be honored
	device, err := b.adapter.Connect(addr, bluetooth.ConnectionParams{ConnectionTimeout: bluetooth.NewDuration(connectionTimeout)})

	if err != nil || device == nil {
		return nil, fmt.Errorf("unable to connect to device: %s", err)
	}

	return device, nil
}

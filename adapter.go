package sphero

import (
	"fmt"

	"github.com/hashicorp/go-hclog"
	"tinygo.org/x/bluetooth"
)

var defaultAdapter = bluetooth.DefaultAdapter

// BluetoothAdapter allows the interaction with the phyiscal bluetooth stack
type BluetoothAdapter struct {
	adapter    *bluetooth.Adapter
	log        hclog.Logger
	scanResult chan ScanResult
}

// ScanResult is returned from the Scan function and encapsulates the
// details of a Bluetooth device
type ScanResult struct {
	Name    string
	Address bluetooth.Addresser
}

func NewBluetoothAdapter(l hclog.Logger) (*BluetoothAdapter, error) {
	err := defaultAdapter.Enable()
	if err != nil {
		l.Error("Unable to enable the bluetooth adapter", "error", err)
		return nil, err
	}

	return &BluetoothAdapter{adapter: defaultAdapter, log: l}, nil
}

// Scan for bluetooth devices, this method returns a channel of ScanResult
// that can be constantly itterated over to print the devices
func (b *BluetoothAdapter) Scan() chan ScanResult {
	b.scanResult = make(chan ScanResult)

	go func() {
		b.adapter.Scan(func(a *bluetooth.Adapter, d bluetooth.ScanResult) {
			name := d.LocalName()
			if name == "" {
				name = "UNKNOWN"
			}

			b.scanResult <- ScanResult{Name: name, Address: d.Address}
		})
	}()

	return b.scanResult
}

func (b *BluetoothAdapter) StopScanning() {
	b.adapter.StopScan()
	close(b.scanResult)
}

func (b *BluetoothAdapter) Connect(addr bluetooth.Addresser) (*bluetooth.Device, error) {
	device, err := b.adapter.Connect(addr, bluetooth.ConnectionParams{})
	if err != nil {
		return nil, fmt.Errorf("unable to connect to device: %s", err)
	}

	return device, nil
}

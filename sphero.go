package sphero

import (
	"fmt"
	"time"

	"github.com/hashicorp/go-hclog"
	"tinygo.org/x/bluetooth"
)

var discoverTimeout = 60 * time.Second

// Sphero defines a type that can communicate with a Sphero Mini over Bluetooth LE
// https://sdk.sphero.com/docs/api_spec/general_api
type Sphero struct {
	device                  *bluetooth.Device
	charAPIV2               bluetooth.DeviceCharacteristic
	charAntiDOS             bluetooth.DeviceCharacteristic
	charDFU                 bluetooth.DeviceCharacteristic
	charDFU2                bluetooth.DeviceCharacteristic
	sequenceNo              int
	log                     hclog.Logger
	outBuffer               []byte
	commandResponse         chan *payload
	streamingResponse       chan *payload
	expectedCommandSequence int
	lastError               error
	backlightEnabled        bool
	next                    func()
}

// NewSphero creates a new Sphero and attempts to connect to the device
// addr can either be supplied as the mac address for the bluetooth device name
//
// example:
//	logger := hclog.Default()
//
//	// create the bluetooth adapter the sphero uses to interface with the computers bluetooth
//	// stack
//	adapter, err := sphero.NewBluetoothAdapter(logger)
//	if err != nil {
//		fmt.Printf("Unable to create a bluetooth adapter: %s\n", err)
//		os.Exit(1)
//	}
//
//	// create the sphero
//	ball, err := sphero.NewSphero(addr, adapter, logger)
//	if err != nil {
//		fmt.Printf("Unable to create a new sphero: %s\n", err)
//		os.Exit(1)
//	}
//
//	// flash the LEDS Red, Green, and Blue
//	ball.
//		SetLEDColor(235, 64, 52).
//		For(1*time.Second).
//		SetLEDColor(52, 235, 88).
//		For(1*time.Second).
//		SetLEDColor(52, 122, 235).
//		For(1 * time.Second)
func NewSphero(addr string, adapter *BluetoothAdapter, l hclog.Logger) (*Sphero, error) {
	var device *bluetooth.Device
	var err error

	// try multiple times as Darwin bluetooth is flakey
	for i := 0; i < 5; i++ {
		l.Debug("Connecting to device", "address", addr, "attempt", i+1)

		device, err = setupConnection(addr, adapter, l)
		if device != nil && err == nil {
			break
		}
	}

	if err != nil || device == nil {
		l.Error("Unable to connect to bluetooth device", "address", addr, "error", err)
		return nil, err
	}

	var services []bluetooth.DeviceService

	// try multiple times as Darwin bluetooth is flakey
	for i := 0; i < 5; i++ {
		l.Debug("Attempting to discover services", "address", addr, "attempt", i+1)

		services, err = device.DiscoverServices([]bluetooth.UUID{})
		if err == nil && len(services) > 0 {
			break
		}
	}

	if err != nil {
		l.Error("Unable to get services for bluetooth device", "address", addr, "error", err)
		return nil, err
	}

	charAPIV2 := getCharacteristic(services, "00010002-574f-4f20-5370-6865726f2121")
	charAntiDOS := getCharacteristic(services, "00020005-574f-4f20-5370-6865726f2121")
	charDFU := getCharacteristic(services, "00020002-574f-4f20-5370-6865726f2121")
	charDFU2 := getCharacteristic(services, "00020004-574f-4f20-5370-6865726f2121")

	// ensure the device does not sleep after 10s
	charAntiDOS.WriteWithoutResponse([]byte("usetheforce...band"))

	s := &Sphero{
		device:      device,
		charAPIV2:   charAPIV2,
		charAntiDOS: charAntiDOS,
		charDFU:     charDFU,
		charDFU2:    charDFU2,
		log:         l,
	}

	s.setup()
	s.blink()

	return s, nil
}

func (s *Sphero) blink() {
	s.
		SetLEDColor(255, 255, 255).
		For(300*time.Millisecond).
		SetLEDColor(255, 255, 255).
		For(300*time.Millisecond).
		SetLEDColor(255, 255, 255).
		For(300*time.Millisecond).
		SetLEDColor(0, 0, 0).
		For(2 * time.Second)
}

func setupConnection(addr string, adapter *BluetoothAdapter, l hclog.Logger) (*bluetooth.Device, error) {
	var bleAddress bluetooth.Addresser

	ac := make(chan bluetooth.Addresser)
	to := time.After(discoverTimeout)

	sr := adapter.Scan()
	defer adapter.StopScanning()

	go func() {
		for r := range sr {
			if r.Name == addr || r.Address.String() == addr {
				ac <- r.Address
			}
		}
	}()

	select {
	case bleAddress = <-ac:
		l.Trace("Found device", "address", addr)
	case <-to:
		return nil, fmt.Errorf("timeout while trying to find device: %s", addr)
	}

	l.Trace("Connecting", "device", addr)

	device, err := adapter.Connect(bleAddress)
	if err != nil {
		l.Trace("Unable to connect to bluetooth device", "address", addr, "error", err)
		return nil, err
	}

	return device, nil
}

func getCharacteristic(ds []bluetooth.DeviceService, uuid string) bluetooth.DeviceCharacteristic {
	uu, err := bluetooth.ParseUUID(uuid)
	if err != nil {
		panic(err)
	}

	for _, s := range ds {
		c, err := s.DiscoverCharacteristics([]bluetooth.UUID{uu})
		if err == nil && len(c) > 0 {
			return c[0]
		}
	}

	panic(fmt.Errorf("characteristic: %s not found", uuid))
}

func (s *Sphero) setup() error {
	s.log.Debug("Setup Sphero")
	s.commandResponse = make(chan *payload)
	s.streamingResponse = make(chan *payload)

	err := s.charAPIV2.EnableNotifications(func(buf []byte) {
		//s.log.Trace("Got response apiv2", "data", buf)

		// if start packet create a new buffer
		if buf[0] == DataPacketStart {
			s.outBuffer = []byte{}
		}

		// increment the buffer
		s.outBuffer = append(s.outBuffer, buf[0])

		// if end packet send to channel
		if buf[0] == DataPacketEnd {
			// construct the payload
			p := &payload{}
			p.decode(s.outBuffer)

			if s.expectedCommandSequence == int(p.Sequence) {
				s.expectedCommandSequence = 0
				s.commandResponse <- p
				return
			}

			s.log.Trace("Got response, disposed", "data", s.outBuffer)
		}

	})

	if err != nil {
		s.log.Error("Unable to receive notifications for charAPIV2", "error", err)
	}

	err = s.charAntiDOS.EnableNotifications(func(buf []byte) {
		s.log.Trace("Got response antidos", "data", buf)
	})

	if err != nil {
		s.log.Error("Unable to receive notifications for charAntiDOS", "error", err)
	}

	err = s.charDFU.EnableNotifications(func(buf []byte) {
		s.log.Trace("Got response dfu", "data", buf)
	})

	if err != nil {
		s.log.Error("Unable to receive notifications for charDFU", "error", err)
	}

	err = s.charDFU2.EnableNotifications(func(buf []byte) {
		s.log.Trace("Got response dfu2", "data", buf)
	})

	if err != nil {
		s.log.Error("Unable to receive notifications for charDFU2", "error", err)
	}

	s.charAntiDOS.WriteWithoutResponse([]byte("usetheforce...band"))

	s.log.Debug("Send Wake")
	s.Wake()

	return nil
}

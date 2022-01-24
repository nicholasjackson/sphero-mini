package sphero

import (
	"fmt"
	"time"

	"github.com/hashicorp/go-hclog"
	"tinygo.org/x/bluetooth"
)

// Sphero protocol
// https://sdk.sphero.com/docs/api_spec/general_api

type Sphero struct {
	charAPIV2               bluetooth.DeviceCharacteristic
	charAntiDOS             bluetooth.DeviceCharacteristic
	charDFU                 bluetooth.DeviceCharacteristic
	charDFU2                bluetooth.DeviceCharacteristic
	sequenceNo              int
	log                     hclog.Logger
	outBuffer               []byte
	commandResponse         chan *Payload
	streamingResponse       chan *Payload
	expectedCommandSequence int
	lastError               error
}

// NewSphero creates a new sphero and attempts to connect to the device
func NewSphero(addr string, adapter *BluetoothAdapter, l hclog.Logger) (*Sphero, error) {
	var bleAddress bluetooth.Addresser

	l.Trace("Discovering device for", "address", addr)

	ac := make(chan bluetooth.Addresser)
	to := time.After(10 * time.Second)

	sr := adapter.Scan()

	go func() {
		for r := range sr {
			if r.Name == addr || r.Address.String() == addr {
				l.Debug("Found device", "addr", addr)
				ac <- r.Address
				adapter.StopScanning()
			}
		}
	}()

	select {
	case bleAddress = <-ac:
		l.Debug("Found device", "address", addr)
	case <-to:
		return nil, fmt.Errorf("timeout while trying to connect to address: %s", addr)
	}

	l.Debug("Connecting", "device", addr)

	//connected := make(chan bool)

	//log.Info("Connecting", "device", addr)
	device, err := adapter.Connect(bleAddress)
	if err != nil {
		l.Error("Unable to connect to bluetooth deivce", "address", addr)
		return nil, err
	}

	services, err := device.DiscoverServices([]bluetooth.UUID{})
	if err != nil {
		l.Error("Unable to get services for bluetooth deivce", "address", addr)
		return nil, err
	}

	charAPIV2 := getCharacteristic(services, "00010002-574f-4f20-5370-6865726f2121")
	charAntiDOS := getCharacteristic(services, "00020005-574f-4f20-5370-6865726f2121")
	charDFU := getCharacteristic(services, "00020002-574f-4f20-5370-6865726f2121")
	charDFU2 := getCharacteristic(services, "00020004-574f-4f20-5370-6865726f2121")

	s := &Sphero{
		charAPIV2:   charAPIV2,
		charAntiDOS: charAntiDOS,
		charDFU:     charDFU,
		charDFU2:    charDFU2,
		log:         l,
	}

	s.setup()

	return s, nil
}

func getCharacteristic(ds []bluetooth.DeviceService, uuid string) bluetooth.DeviceCharacteristic {
	uu, err := bluetooth.ParseUUID(uuid)
	if err != nil {
		panic(err)
	}

	for _, s := range ds {
		c, err := s.DiscoverCharacteristics([]bluetooth.UUID{uu})
		if err == nil {
			return c[0]
		}
	}

	panic(fmt.Errorf("characteristic: %s not found", uuid))
}

func (s *Sphero) setup() error {
	s.log.Debug("Setup Sphero")
	s.commandResponse = make(chan *Payload)
	s.streamingResponse = make(chan *Payload)

	s.charAPIV2.EnableNotifications(func(buf []byte) {
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
			p := &Payload{}
			p.Decode(s.outBuffer)

			if s.expectedCommandSequence == int(p.Sequence) {
				s.expectedCommandSequence = 0
				s.commandResponse <- p
				return
			}

			s.log.Trace("Got response, disposed", "data", s.outBuffer)
		}

	})

	s.charAntiDOS.EnableNotifications(func(buf []byte) {
		s.log.Trace("Got response antidos", "data", buf)
	})

	s.charDFU.EnableNotifications(func(buf []byte) {
		s.log.Trace("Got response dfu", "data", buf)
	})

	s.charDFU2.EnableNotifications(func(buf []byte) {
		s.log.Trace("Got response dfu2", "data", buf)
	})

	s.charAntiDOS.WriteWithoutResponse([]byte("usetheforce...band"))

	s.log.Debug("Send Wake")
	s.Wake()

	return nil
}

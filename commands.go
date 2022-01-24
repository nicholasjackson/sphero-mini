package sphero

import (
	"fmt"
	"time"

	"tinygo.org/x/bluetooth"
)

func (s *Sphero) Wait(d time.Duration) *Sphero {
	time.Sleep(d)

	return s
}

func (s *Sphero) GetLastError() error {
	return s.lastError
}

// Wake brings the device out of sleep mode
func (s *Sphero) Wake() *Sphero {
	s.log.Debug("Wake")
	_, err := s.send(s.charAPIV2, DevicePowerInfo, PowerCommandsWake, true, []byte{})
	if err != nil {
		s.log.Error("unable to wake sphero", "error", err)
		s.lastError = err
	}

	return s
}

func (s *Sphero) Sleep() *Sphero {
	s.log.Debug("Sleep")
	_, err := s.send(s.charAPIV2, DevicePowerInfo, PowerCommandsSleep, true, []byte{})
	if err != nil {
		s.log.Error("unable to sleep sphero", "error", err)
		s.lastError = err
	}

	return s
}

func (s *Sphero) GetBatteryVoltage() *Sphero {
	s.log.Debug("GetBatteryVoltage")
	_, err := s.send(s.charAPIV2, DevicePowerInfo, PowerCommandsBatteryVoltage, true, []byte{})

	if err != nil {
		s.log.Error("unable to get battery voltage", "error", err)
		s.lastError = err
	}

	return s
}

func (s *Sphero) SetLEDColor(r, g, b uint8) *Sphero {
	s.log.Debug("Enabling LED", "r", r, "g", g, "b", b)

	payload := []byte{0x00, 0x0e, r, g, b}

	_, err := s.send(s.charAPIV2, DeviceUserIO, UserIOCommandsAllLEDs, true, payload)
	if err != nil {
		s.log.Error("unable to set LED color", "error", err)
		s.lastError = err
	}

	return s
}

// https://github.com/MProx/Sphero_mini/blob/1dea6ff7f59260ea5ecee9cb9a7c9f46f1f8a6d9/sphero_mini.py#L243
func (s *Sphero) send(dc bluetooth.DeviceCharacteristic, deviceID, commandID byte, expectResponse bool, payload []byte) (*Payload, error) {
	// sequence ensures we can associate a request with a response
	s.sequenceNo += 1
	if s.sequenceNo > 255 {
		s.sequenceNo = 0
	}

	//FlagResetsInactivityTimeout + FlagRequestsResponse

	// are we expecting a response
	if expectResponse {
		s.expectedCommandSequence = s.sequenceNo
	}

	// define the header for the send request
	p := Payload{
		Flags:    FlagResetsInactivityTimeout + FlagRequestsResponse, // set the flags
		DeviceID: deviceID,                                           // send is for the given device id
		Command:  commandID,                                          // with the command
		Sequence: byte(s.sequenceNo),                                 // set the sequence id to ensure that packets are orderable
		Payload:  payload,
	}

	data := p.Encode()

	s.log.Trace("Sending data", "bytes", data)

	_, err := dc.WriteWithoutResponse(data)
	if err != nil {
		s.log.Error("Error sending data")
		return nil, err
	}

	if !expectResponse {
		return nil, nil
	}

	// wait for response
	timeout := time.After(10 * time.Second)
	select {
	case <-timeout:
		s.log.Error("Timeout waiting for response")
		return nil, fmt.Errorf("Timeout waiting for data")
	case p := <-s.commandResponse:
		s.log.Debug("Got response", "data", p)
		return p, nil
	}
}

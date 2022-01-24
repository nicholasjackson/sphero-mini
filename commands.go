package sphero

import (
	"fmt"
	"time"

	"tinygo.org/x/bluetooth"
)

func (s *Sphero) For(d time.Duration) *Sphero {
	time.Sleep(d)

	if s.next != nil {
		s.next()
		s.next = nil
	}

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

	if s.backlightEnabled {
		s.DisableBackLight()
	}

	defer s.device.Disconnect()

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
	s.next = func() {
		// TODO: if the next call is set LED do not turn off as the transition is smoother
		s.setLEDColor(0, 0, 0)
	}

	s.setLEDColor(r, g, b)

	return s
}

func (s *Sphero) setLEDColor(r, g, b uint8) *Sphero {
	s.log.Debug("Enabling LED", "r", r, "g", g, "b", b)

	payload := []byte{0x00, 0x0e, r, g, b}

	_, err := s.send(s.charAPIV2, DeviceUserIO, UserIOCommandsAllLEDs, true, payload)
	if err != nil {
		s.log.Error("unable to set LED color", "error", err)
		s.lastError = err
	}

	s.next = func() {
		// TODO: if the next call is set LED do not turn off as the transition is smoother
		s.SetLEDColor(0, 0, 0)
	}

	return s
}

func (s *Sphero) EnableBackLight() *Sphero {
	s.log.Debug("Set backlight LED")
	s.backlightEnabled = true

	payload := []byte{0x00, 0x01, 255}

	_, err := s.send(s.charAPIV2, DeviceUserIO, UserIOCommandsAllLEDs, true, payload)
	if err != nil {
		s.log.Error("unable to set LED backlight", "error", err)
		s.lastError = err
	}

	return s
}

func (s *Sphero) DisableBackLight() *Sphero {
	s.log.Debug("Disable backlight LED")
	s.backlightEnabled = false

	payload := []byte{0x00, 0x01, 0}

	_, err := s.send(s.charAPIV2, DeviceUserIO, UserIOCommandsAllLEDs, true, payload)
	if err != nil {
		s.log.Error("unable to set LED backlight", "error", err)
		s.lastError = err
	}

	return s
}

// Roll towards heading given in degrees 0-360 at speed as an integer 0-255
func (s *Sphero) Roll(heading, speed int) *Sphero {
	s.next = func() {
		s.roll(0, 1)
		// give the ball time to stop before changing direction
		time.Sleep(500 * time.Millisecond)
	}

	s.roll(heading, speed)

	return s
}

func (s *Sphero) roll(heading, speed int) {
	s.log.Debug("Roll", "heading", heading, "speed", speed)

	speedH := uint8((speed & 0xFF00) >> 8)
	speedL := uint8(speed & 0xFF)
	headingH := uint8((heading & 0xFF00) >> 8)
	headingL := uint8(heading & 0xFF)

	payload := []byte{speedL, headingH, headingL, speedH}

	_, err := s.send(s.charAPIV2, DeviceDriving, DrivingCommandsWithHeading, true, payload)
	if err != nil {
		s.log.Error("unable to Roll in direction", "heading", heading, "speed", speed, "error", err)
		s.lastError = err
	}
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

package main

import (
	"fmt"
	"time"

	"github.com/hashicorp/go-hclog"
	"tinygo.org/x/bluetooth"
)

// Sphero protocol
// https://sdk.sphero.com/docs/api_spec/general_api

const (
	DataPacketStart = 0x8D
	DataPacketEnd   = 0xD8

	FlagIsResponse                = 0x01
	FlagRequestsResponse          = 0x02
	FlagRequestsOnlyErrorResponse = 0x04
	FlagResetsInactivityTimeout   = 0x08

	DevicePowerInfo = 0x13
	DeviceUserIO    = 0x1a

	PowerCommandsDeepSleep      = 0x00
	PowerCommandsSleep          = 0x01
	PowerCommandsBatteryVoltage = 0x03
	PowerCommandsWake           = 0x0D

	UserIOCommandsAllLEDs = 0x0e

	SystemInfoCommandsBootLoaderVersion = 0x01
)

type Sphero struct {
	charAPIV2      bluetooth.DeviceCharacteristic
	charAntiDOS    bluetooth.DeviceCharacteristic
	charDFU        bluetooth.DeviceCharacteristic
	charDFU2       bluetooth.DeviceCharacteristic
	sequenceNo     int
	log            hclog.Logger
	outBuffer      []byte
	outData        chan []byte
	expectResponse bool
}

func (s *Sphero) Setup() error {
	s.log.Debug("Setup Sphero")
	s.outData = make(chan []byte)

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
			if s.expectResponse {
				s.outData <- s.outBuffer
				return
			}

			s.log.Trace("Got response, disposed", "data", s.outBuffer)
			s.expectResponse = false
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

// Wake brings the device out of sleep mode
func (s *Sphero) Wake() error {
	_, err := s.send(s.charAPIV2, DevicePowerInfo, PowerCommandsWake, false, []byte{})

	return err
}

func (s *Sphero) Sleep() error {
	_, err := s.send(s.charAPIV2, DevicePowerInfo, PowerCommandsSleep, false, []byte{})

	return err
}

func (s *Sphero) GetBatteryVoltage() error {
	_, err := s.send(s.charAPIV2, DevicePowerInfo, PowerCommandsBatteryVoltage, true, []byte{})

	return err
}

func (s *Sphero) SetLEDColor(r, g, b uint8) error {
	s.log.Debug("Enabling LED", "r", r, "g", g, "b", b)

	payload := []byte{0x00, 0x0e, r, g, b}

	_, err := s.send(s.charAPIV2, DeviceUserIO, UserIOCommandsAllLEDs, true, payload)

	return err
}

// https://github.com/MProx/Sphero_mini/blob/1dea6ff7f59260ea5ecee9cb9a7c9f46f1f8a6d9/sphero_mini.py#L243
func (s *Sphero) send(dc bluetooth.DeviceCharacteristic, deviceID, commandID byte, expectResponse bool, payload []byte) ([]byte, error) {
	// sequence ensures we can associate a request with a response
	s.sequenceNo += 1
	if s.sequenceNo > 255 {
		s.sequenceNo = 0
	}

	// are we expecting a response
	s.expectResponse = expectResponse

	// define the header for the send request
	sendBytes := []byte{
		DataPacketStart, // first byte is always 0x08
		FlagResetsInactivityTimeout + FlagRequestsResponse, // set the flags
		deviceID,           // send is for the given device id
		commandID,          // with the command
		byte(s.sequenceNo), // set the sequence id to ensure that packets are orderable
	}

	// add the payload
	sendBytes = append(sendBytes, payload...)

	// add the end of the request checksum and end byte
	sendBytes = append(sendBytes, calculateChecksum(sendBytes), DataPacketEnd)

	s.log.Trace("Sending data", "bytes", sendBytes)

	_, err := dc.WriteWithoutResponse(sendBytes)
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
	case buf := <-s.outData:
		s.log.Debug("Got response", "data", buf)
		return buf, nil
	}

	return nil, nil
}

func calculateChecksum(b []byte) uint8 {
	checksum := uint16(0)
	for _, b := range b[1:] {
		checksum = checksum + uint16(b)
	}

	return uint8(^(checksum % 256))
}

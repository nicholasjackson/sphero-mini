package main

import (
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
	charAPIV2   bluetooth.DeviceCharacteristic
	charAntiDOS bluetooth.DeviceCharacteristic
	charDFU     bluetooth.DeviceCharacteristic
	charDFU2    bluetooth.DeviceCharacteristic
	sequenceNo  int
	log         hclog.Logger
}

func (s *Sphero) Setup() error {
	s.log.Debug("Setup Sphero")

	s.charAPIV2.EnableNotifications(func(buf []byte) {
		s.log.Trace("Got response apiv2", "data", buf)
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
	return s.send(s.charAPIV2, DevicePowerInfo, PowerCommandsWake, []byte{})
}

func (s *Sphero) Sleep() error {
	return s.send(s.charAPIV2, DevicePowerInfo, PowerCommandsSleep, []byte{})
}

func (s *Sphero) GetBatteryVoltage() error {
	return s.send(s.charAPIV2, DevicePowerInfo, PowerCommandsBatteryVoltage, []byte{})
}

func (s *Sphero) SetLEDColor(r, g, b uint8) error {
	s.log.Debug("Enabling LED", "r", r, "g", g, "b", b)

	payload := []byte{0x00, 0x0e, r, g, b}
	return s.send(s.charAPIV2, DeviceUserIO, UserIOCommandsAllLEDs, payload)
}

// https://github.com/MProx/Sphero_mini/blob/1dea6ff7f59260ea5ecee9cb9a7c9f46f1f8a6d9/sphero_mini.py#L243
func (s *Sphero) send(dc bluetooth.DeviceCharacteristic, deviceID, commandID byte, payload []byte) error {
	// sequence ensures we can associate a request with a response
	s.sequenceNo += 1
	if s.sequenceNo > 255 {
		s.sequenceNo = 0
	}

	dc.EnableNotifications(func(buf []byte) {
		s.log.Trace("Got response", "characteristics", dc.UUID, "data", buf)
	})

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

	_, err := dc.WriteWithoutResponse(sendBytes)
	if err != nil {
		s.log.Error("Error sending data")
		return err
	}

	s.log.Trace("Sending data", "bytes", sendBytes)

	return nil
}

func calculateChecksum(b []byte) uint8 {
	checksum := uint16(0)
	for _, b := range b[1:] {
		checksum = checksum + uint16(b)
	}

	return uint8(^(checksum % 256))
}

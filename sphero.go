package main

import (
	"github.com/hashicorp/go-hclog"
	"tinygo.org/x/bluetooth"
)

const (
	DataPacketStart = 0x8D
	DataPacketEnd   = 0x8D

	FlagIsResponse                = 0x01
	FlagRequestsResponse          = 0x02
	FlagRequestsOnlyErrorResponse = 0x04
	FlagResetsInactivityTimeout   = 0x08

	DevicePowerInfo = 0x13

	PowerCommandsWake = 0x0D
)

type Sphero struct {
	charAPIV2   bluetooth.DeviceCharacteristic
	charAntiDOS bluetooth.DeviceCharacteristic
	charDFU     bluetooth.DeviceCharacteristic
	charDFU2    bluetooth.DeviceCharacteristic
	log         hclog.Logger
}

// Wake brings the device out of sleep mode
func (s *Sphero) Wake() error {
	return s.send(s.charAPIV2, DevicePowerInfo, PowerCommandsWake)
}

// https://github.com/MProx/Sphero_mini/blob/1dea6ff7f59260ea5ecee9cb9a7c9f46f1f8a6d9/sphero_mini.py#L243
func (s *Sphero) send(dc bluetooth.DeviceCharacteristic, deviceID, commandID byte) error {
	sequenceNo := 0

	dc.EnableNotifications(func(buf []byte) {
		s.log.Trace("Got response", "characteristics", "charAPIV2", "data", buf)
	})

	// define the header for the send request
	sendBytes := []byte{
		DataPacketStart, // first byte is always 0x08
		FlagResetsInactivityTimeout + FlagRequestsResponse, // set the flags
		deviceID,         // send is for the given device id
		commandID,        // with the command
		byte(sequenceNo), // set the sequence id to ensure that packets are orderable
	}

	checksum := 0
	for _, b := range sendBytes[1:] {
		checksum = checksum + int(b)&0x00
	}

	sendBytes = append(sendBytes, byte(checksum), DataPacketEnd)

	_, err := dc.WriteWithoutResponse(sendBytes)
	if err != nil {
		s.log.Error("Error sending data")
		return err
	}

	s.log.Trace("Sending data", "bytes", sendBytes)

	return nil
}

package sphero

import "fmt"

type Payload struct {
	Flags    uint8
	DeviceID uint8
	Command  uint8
	Sequence uint8
	Error    uint8
	Payload  []byte
}

func (p *Payload) Encode() []byte {
	sendBytes := []byte{
		DataPacketStart, // first byte is always 0x08
		p.Flags,         // set the flags
		p.DeviceID,      // send is for the given device id
		p.Command,       // with the command
		p.Sequence,      // set the sequence id to ensure that packets are orderable
	}

	// add the payload
	sendBytes = append(sendBytes, p.Payload...)

	// calculateChecksum
	cs := calculateChecksum(sendBytes)

	sendBytes = append(sendBytes, cs, DataPacketEnd)

	return sendBytes
}

func (p *Payload) Decode(d []byte) error {
	p.Flags = d[1]
	p.DeviceID = d[2]
	p.Command = d[3]
	p.Sequence = d[4]
	p.Error = d[5]

	checksum := d[6]

	// compare checksum
	cc := calculateChecksum(d[1 : len(d)-1])

	if checksum != cc {
		return fmt.Errorf("checksum invalid")
	}

	if len(d) > 7 {
		p.Payload = d[7 : len(d)-2]
	}

	return nil
}

func calculateChecksum(b []byte) uint8 {
	checksum := uint16(0)
	for _, b := range b[1:] {
		checksum = checksum + uint16(b)
	}

	return uint8(^(checksum % 256))
}

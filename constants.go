package sphero

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

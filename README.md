# Sphero Mini API for Go

This repository contains a Go package for interfacing with the [Sphero Mini](https://sphero.com/products/sphero-mini) using Bluetooth LE.

## Example

To interact with the Sphero Mini you need to connect to it using either the devices short name e.g. `SM-BA93` or the 
mac address for the device e.g. `15:DB:E2:E1:D1:77`. The following simple example shows how the `BluetoothAdapter`
can be used for this function.

**Scanning bluetooth devices:**

```go
ad, err := sphero.NewBluetoothAdapter(createLogger())
if err != nil {
	fmt.Printf("Unable to create a bluetooth adapter: %s\n", err)
	os.Exit(1)
}

sr := ad.Scan()

fmt.Printf("%-30s %s\n", "Name", "Mac Address")
fmt.Printf("%-30s %s\n", "-----------------------------", "-----------------")
for r := range sr {
	fmt.Printf("%-30s %s\n", r.Name, r.Address.String())
}
```

**Interacting with the Sphero**

```go
// create a new logger
logger := createLogger()

// create a bluetooth adapter
adapter, err := sphero.NewBluetoothAdapter(logger)
if err != nil {
	fmt.Printf("Unable to create a bluetooth adapter: %s\n", err)
	os.Exit(1)
}

// create a new sphero using the Bluetooth short name for the peripheral
ball, err := sphero.NewSphero("SM-34ED", adapter, logger)
if err != nil {
	fmt.Printf("Unable to create a new sphero: %s\n", err)
	os.Exit(1)
}

// Put the ball to sleep, you should always call this method to save battery
defer ball.Sleep()

// enable the backlight, this is useful to see which direction the sphero is headed
ball.EnableBackLight()
time.Sleep(5 * time.Second)

// flash the LED Red, Green, and Blue for 1 second each 
ball.
	SetLEDColor(235, 64, 52).For(1*time.Second).
	SetLEDColor(52, 235, 88).For(1*time.Second).
	SetLEDColor(52, 122, 235).For(1 * time.Second)

time.Sleep(5 * time.Second)

// Roll forward for 1 second, wait 1 second, and then roll back for 1 second
ball.
  Roll(0, 150).For(1*time.Second).
  Wait().For(1*time.Second).
	Roll(180, 150).For(1 * time.Second)

```

## Bluetooth Support

Bluetooth support is limited to that provided by [TinyGo Bluetooth](https://github.com/tinygo-org/bluetooth) 
at present this support covers the following platforms.

* Mac Intel - Supported but curently unstable
* Mac Arm64 (M1) - Unknown
* Linux Intel - Supported
* Linux Arm64 - Supported
* Windows - Not supported

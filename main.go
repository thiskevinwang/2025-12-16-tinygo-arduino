package main

import (
	"machine"
	"time"

	"tinygo.org/x/bluetooth"
)

var (
	// TODO: resolve type error:
	// undefined: bluetooth.DefaultAdapter [linux,arm]
	adapter = bluetooth.DefaultAdapter

	heartRateMeasurement bluetooth.Characteristic
	bodyLocation         bluetooth.Characteristic
	controlPoint         bluetooth.Characteristic

	heartRate uint8 = 75 // 75bpm
)

func scaleADCToDivisor(adcValue uint16) uint16 {
	if adcValue > 60000 {
		return 20 // 1/20 second
	} else {
		return 2 // 1/2 second
	}
}

// XIAO Seeed Studio
// nRF52840 https://www.seeedstudio.com/Seeed-XIAO-BLE-nRF52840-p-5201.html
//
// TinyGo: https://tinygo.org/docs/reference/microcontrollers/xiao-ble/#interfaces
func main() {
	println("starting")
	must("enable BLE stack", adapter.Enable())
	adv := adapter.DefaultAdvertisement()

	must("config adv", adv.Configure(bluetooth.AdvertisementOptions{
		LocalName:    "Progressor FAKE",
		ServiceUUIDs: []bluetooth.UUID{bluetooth.ServiceUUIDHeartRate},
	}))

	// Add service BEFORE starting advertisement
	must("add service", adapter.AddService(&bluetooth.Service{
		UUID: bluetooth.ServiceUUIDHeartRate,
		Characteristics: []bluetooth.CharacteristicConfig{
			{
				Handle: &heartRateMeasurement,
				UUID:   bluetooth.CharacteristicUUIDHeartRateMeasurement,
				Value:  []byte{0, heartRate},
				Flags:  bluetooth.CharacteristicNotifyPermission,
			},
			{
				Handle: &bodyLocation,
				UUID:   bluetooth.CharacteristicUUIDBodySensorLocation,
				Value:  []byte{1}, // "Chest"
				Flags:  bluetooth.CharacteristicReadPermission,
			},
			{
				Handle: &controlPoint,
				UUID:   bluetooth.CharacteristicUUIDHeartRateControlPoint,
				Value:  []byte{0},
				Flags:  bluetooth.CharacteristicWritePermission,
			},
		},
	}))

	// Start advertising AFTER service is registered
	must("start adv", adv.Start())

	/// configure analog-to-digital converter (ADC) for light sensor
	machine.InitADC()                   // https://github.com/aykevl/board/blob/a919e54134677344aaee1dde53eb629377614259/board-pybadge.go#L37
	adc := machine.ADC{Pin: machine.A5} // <-- use A0..A5 on XIAO BLE
	adc.Configure(machine.ADCConfig{})

	// RGB LED pins (Common Cathode)
	red := machine.D0
	green := machine.D1
	blue := machine.D2

	red.Configure(machine.PinConfig{Mode: machine.PinOutput})
	green.Configure(machine.PinConfig{Mode: machine.PinOutput})
	blue.Configure(machine.PinConfig{Mode: machine.PinOutput})

	onOffButton := machine.D7
	onOffButton.Configure(machine.PinConfig{Mode: machine.PinInputPullup})

	checkInterval := 10 * time.Millisecond

	// Helper function that sleeps but checks button frequently
	// and sleeps dynamically based the input.
	sleepWithButtonCheck := func(divisor uint16) bool {
		// Avoid division by zero
		if divisor == 0 {
			divisor = 1
		}

		elapsed := time.Duration(0)
		for elapsed < (time.Second / time.Duration(divisor)) {
			if !onOffButton.Get() {
				return true // Button pressed
			}
			time.Sleep(checkInterval)
			elapsed += checkInterval
		}
		return false
	}

	// give some time to set up measurement environment
	time.Sleep(2 * time.Second)
	println("[debug]")
	println("uuid:", bluetooth.ServiceUUIDHeartRate.String())
	println("uuid:", bluetooth.ServiceUUIDHeartRate.String())

	// Main loop: cycle through colors, but turn white if button pressed
	for {
		println("[device]", "cycling colors")

		if !onOffButton.Get() {
			// Button pressed, turn make led white
			red.High()
			green.High()
			blue.High()
			time.Sleep(200 * time.Millisecond)
			red.Low()
			green.Low()
			blue.Low()
			time.Sleep(100 * time.Millisecond)
			continue // Skip the color cycle when button is pressed
		}

		// Red
		red.High()
		green.Low()
		blue.Low()
		if sleepWithButtonCheck(scaleADCToDivisor(adc.Get())) {
			continue
		}

		// Magenta (Red + Blue)
		red.High()
		green.Low()
		blue.High()
		if sleepWithButtonCheck(scaleADCToDivisor(adc.Get())) {
			continue
		}

		// Blue
		red.Low()
		green.Low()
		blue.High()
		if sleepWithButtonCheck(scaleADCToDivisor(adc.Get())) {
			continue
		}

		// Cyan (Green + Blue)
		red.Low()
		green.High()
		blue.High()
		if sleepWithButtonCheck(scaleADCToDivisor(adc.Get())) {
			continue
		}

		// Green
		red.Low()
		green.High()
		blue.Low()
		if sleepWithButtonCheck(scaleADCToDivisor(adc.Get())) {
			continue
		}

		// Yellow (Red + Green)
		red.High()
		green.High()
		blue.Low()
		if sleepWithButtonCheck(scaleADCToDivisor(adc.Get())) {
			continue
		}
	}
}

// Findings
// ------------------------------------------------
// adc.Get() returns ~ 4000 if nothing is connected.
//
// adc.Get() with voltage splitter w/
// - LDR
// - 10kΩ resistor to GND
// returns ~ 60,000 in bright light
// returns ~ 30,000 in darkness

// Education
// ------------------------------------------------
// The adc.Get() function returns a 16-bit unsigned integer
// (uint16) representing the analog voltage on the pin,
// scaled to a range of 0 to 65535.

func must(action string, err error) {
	if err != nil {
		panic("failed to " + action + ": " + err.Error())
	}
}

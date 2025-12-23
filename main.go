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

func log(mod, msg any) {
	now := time.Now()
	println(now.UnixMilli(), mod, msg)
}

// XIAO Seeed Studio
// nRF52840 https://www.seeedstudio.com/Seeed-XIAO-BLE-nRF52840-p-5201.html
//
// TinyGo: https://tinygo.org/docs/reference/microcontrollers/xiao-ble/#interfaces
func main() {
	log("[main]", "starting")
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
	machine.InitADC()
	adc := machine.ADC{Pin: machine.A5} // <-- use A0..A5 on XIAO BLE
	adc.Configure(machine.ADCConfig{})

	// RGB LED pins (Common Cathode)
	red := machine.D0
	green := machine.D1
	blue := machine.D2

	red.Configure(machine.PinConfig{Mode: machine.PinOutput})
	green.Configure(machine.PinConfig{Mode: machine.PinOutput})
	blue.Configure(machine.PinConfig{Mode: machine.PinOutput})

	// Ensure LEDs are off by default
	red.Low()
	green.Low()
	blue.Low()

	onOffButton := machine.D6
	onOffButton.Configure(machine.PinConfig{Mode: machine.PinInputPullup})

	// HiLetgo HC-SR501 - PIR Sensor pins
	// Note: PIR VCC is connected to the XIAO 5V pin
	pirOUT := machine.D7
	pirOUT.Configure(machine.PinConfig{Mode: machine.PinInput})

	// PIR sensor warm-up period
	log("[device]", "waiting")
	time.Sleep(5 * time.Second)
	log("[device]", "PIR sensor ready")

	// Main loop
	for {
		// 1. Read + log ADC (Light dependent resistor)
		adcValue := adc.Get()
		log("[adc]", adcValue)

		// 2. Log Button - log-only
		if !onOffButton.Get() {
			log("[button]", "pressed")
		}

		// 3. PIR takes precedence. If no motion, set LED based on ADC thresholds.
		if pirOUT.Get() {
			log("[pir]", "motion detected!")
			// Light up all LEDs (White)
			red.High()
			green.High()
			blue.High()
		} else {
			if adcValue > 50000 {
				// Blue
				red.Low()
				green.Low()
				blue.High()
			} else if adcValue > 40000 {
				// Green
				red.Low()
				green.High()
				blue.Low()
			} else {
				// Red
				red.High()
				green.Low()
				blue.Low()
			}
		}

		time.Sleep(500 * time.Millisecond)
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
	} else {
		log("[must]", action+" OK")
	}
}

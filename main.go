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

	motionHeartbeat bluetooth.Characteristic
	toggle          bluetooth.Characteristic
	controlPoint    bluetooth.Characteristic

	extLedOn bool

	// Mirror of extLedOn exposed via the read characteristic.
	toggleValue [1]byte
	toggleDirty bool

	isAdvertising bool
	isConnected   bool
)

func boolByte(v bool) byte {
	if v {
		return 1
	}
	return 0
}

// parseExtLedOnWrite parses a write payload intended to control extLedOn.
//
// Supported payloads (no heap allocations):
// - Hex/raw bytes: 0x00 = OFF, 0x01 = ON
// - ASCII text: "0" = OFF, "1" = ON
// - ASCII text: "on"/"off" (case-insensitive)
//
// Notes:
// - Only offset==0 is supported (single write).
// - Other values are ignored (ok=false).
func parseExtLedOnWrite(offset int, value []byte) (on bool, ok bool) {
	if offset != 0 || len(value) == 0 {
		return false, false
	}

	if len(value) == 1 {
		switch value[0] {
		case 0x00, '0':
			return false, true
		case 0x01, '1':
			return true, true
		default:
			return false, false
		}
	}

	if len(value) == 2 {
		b0 := value[0]
		b1 := value[1]
		if (b0 == 'o' || b0 == 'O') && (b1 == 'n' || b1 == 'N') {
			return true, true
		}
		return false, false
	}

	if len(value) == 3 {
		b0 := value[0]
		b1 := value[1]
		b2 := value[2]
		if (b0 == 'o' || b0 == 'O') && (b1 == 'f' || b1 == 'F') && (b2 == 'f' || b2 == 'F') {
			return false, true
		}
		return false, false
	}

	return false, false
}

func log(mod, msg any) {
	now := time.Now()
	println(now.UnixMilli(), mod, msg)
}

func must(action string, err error) {
	if err != nil {
		// Red LED on error (Active Low)
		machine.LED_RED.Low()
		panic("failed to " + action + ": " + err.Error())
	} else {
		log("[must]", action+" OK")
	}
}

// XIAO Seeed Studio
// nRF52840 https://www.seeedstudio.com/Seeed-XIAO-BLE-nRF52840-p-5201.html
//
// TinyGo: https://tinygo.org/docs/reference/microcontrollers/xiao-ble/#interfaces
func main() {
	// Initialize onboard LEDs (Active Low)
	machine.LED_RED.Configure(machine.PinConfig{Mode: machine.PinOutput})
	machine.LED_GREEN.Configure(machine.PinConfig{Mode: machine.PinOutput})
	machine.LED_BLUE.Configure(machine.PinConfig{Mode: machine.PinOutput})
	machine.LED_RED.High()
	machine.LED_GREEN.High()
	machine.LED_BLUE.High()

	log("[main]", "starting")
	// Initialize read characteristic to current ext LED state.
	toggleValue[0] = boolByte(extLedOn)
	toggleDirty = false
	must("enable BLE stack", adapter.Enable())

	adapter.SetConnectHandler(func(device bluetooth.Device, connected bool) {
		isConnected = connected
		if connected {
			isAdvertising = false
			println("BLE: connected")
		} else {
			isAdvertising = true
			println("BLE: disconnected")
			// Restart advertising on disconnect
			adapter.DefaultAdvertisement().Start()
		}
	})

	adv := adapter.DefaultAdvertisement()

	must("config adv", adv.Configure(bluetooth.AdvertisementOptions{
		LocalName:    "TinyGo Device",
		ServiceUUIDs: []bluetooth.UUID{bluetooth.ServiceUUIDHeartRate},
	}))

	// Add service BEFORE starting advertisement
	must("add service", adapter.AddService(&bluetooth.Service{
		UUID: bluetooth.ServiceUUIDHeartRate,
		Characteristics: []bluetooth.CharacteristicConfig{
			{
				Handle: &motionHeartbeat,
				UUID:   bluetooth.CharacteristicUUIDHeartRateMeasurement,
				Value:  []byte{0},
				Flags:  bluetooth.CharacteristicNotifyPermission,
			},
			{
				Handle: &toggle,
				UUID:   bluetooth.CharacteristicUUIDBodySensorLocation,
				Value:  toggleValue[:],
				Flags:  bluetooth.CharacteristicReadPermission,
			},
			{
				// Write to control ext LED (supports both Hex and Text clients):
				// - Hex: 00 = OFF, 01 = ON
				// - Text: "0"/"1" or "off"/"on" (case-insensitive)
				Handle: &controlPoint,
				UUID:   bluetooth.CharacteristicUUIDHeartRateControlPoint,
				Flags:  bluetooth.CharacteristicWritePermission | bluetooth.CharacteristicWriteWithoutResponsePermission,
				WriteEvent: func(client bluetooth.Connection, offset int, value []byte) {
					if on, ok := parseExtLedOnWrite(offset, value); ok {
						// if led on and received ON command, do nothing
						if extLedOn && on {
							return
						}
						// if led off and received OFF command, do nothing
						if !extLedOn && !on {
							return
						}

						extLedOn = on
						toggleValue[0] = boolByte(extLedOn)
						toggleDirty = true

						// Note: TinyGo does not allow heap allocations (like creating strings
						// or boxing values into interfaces) inside interrupts to ensure
						// deterministic behavior and prevent memory corruption.
						// println("BLE: write offset", offset, "value", value[0])
					}
				},
			},
		},
	}))

	// Start advertising AFTER service is registered
	must("start adv", adv.Start())
	isAdvertising = true

	/// configure analog-to-digital converter (ADC) for light sensor
	machine.InitADC()
	adc := machine.ADC{Pin: machine.A5} // <-- use A0..A5 on XIAO BLE
	adc.Configure(machine.ADCConfig{})

	// RGB LED pins (Common Cathode)
	ext_led_red := machine.D0
	ext_led_grn := machine.D1
	ext_led_blu := machine.D2

	ext_led_red.Configure(machine.PinConfig{Mode: machine.PinOutput})
	ext_led_grn.Configure(machine.PinConfig{Mode: machine.PinOutput})
	ext_led_blu.Configure(machine.PinConfig{Mode: machine.PinOutput})

	// Ensure LEDs are off by default
	ext_led_red.Low()
	ext_led_grn.Low()
	ext_led_blu.Low()

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

	var lastMotion bool

	// Main loop
	for {
		// Keep the read characteristic in sync with extLedOn.
		if toggleDirty {
			toggleDirty = false
			toggle.Write(toggleValue[:])
		}

		// 1. Read + log ADC (Light dependent resistor)
		adcValue := adc.Get()
		log("[adc]", adcValue)

		// 2. Log Button - log-only
		if !onOffButton.Get() {
			log("[button]", "pressed")
		}

		// 3. Update Status LEDs
		if isConnected {
			// Blue = connected
			machine.LED_RED.High()
			machine.LED_GREEN.High()
			machine.LED_BLUE.Low()
		} else if isAdvertising {
			// Green = advertising
			machine.LED_RED.High()
			machine.LED_GREEN.Low()
			machine.LED_BLUE.High()
		} else {
			// Off
			machine.LED_RED.High()
			machine.LED_GREEN.High()
			machine.LED_BLUE.High()
		}

		// 4. PIR logging and BLE notification
		motion := pirOUT.Get()
		if motion != lastMotion {
			lastMotion = motion
			if motion {
				log("[pir]", "motion detected!")
				motionHeartbeat.Write([]byte{1})
			} else {
				log("[pir]", "motion stopped")
				motionHeartbeat.Write([]byte{0})
			}
		}

		// 5. External LED control
		if !extLedOn {
			ext_led_red.Low()
			ext_led_grn.Low()
			ext_led_blu.Low()
		} else {
			if motion {
				// Light up all LEDs (White)
				ext_led_red.High()
				ext_led_grn.High()
				ext_led_blu.High()
			} else {
				if adcValue > 50000 {
					// Blue
					ext_led_red.Low()
					ext_led_grn.Low()
					ext_led_blu.High()
				} else if adcValue > 40000 {
					// Green
					ext_led_red.Low()
					ext_led_grn.High()
					ext_led_blu.Low()
				} else {
					// Red
					ext_led_red.High()
					ext_led_grn.Low()
					ext_led_blu.Low()
				}
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

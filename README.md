This is TinyGo project for a XIAO BLE nRF52840 board.

```bash
tinygo flash -target=xiao-ble .
tinygo monitor
```

```bash
tinygo flash -monitor -target=xiao-ble . 
```

```bash
tinygo build -o firmware.uf2 -target=xiao-ble .
```

> My XIAO is connected to my mac. How do I find it?

```console
user@~: $ ls /Volumes/ && echo "---Serial Ports---" && ls /dev/cu.* 2>/dev/null || echo "No serial ports found"

Macintosh HD
---Serial Ports---
/dev/cu.Bluetooth-Incoming-Port /dev/cu.debug-console           /dev/cu.usbmodem101
```

Note: The XIAO BLE only shows up as a mounted volume (in Volumes) when it's in bootloader mode (double-tap reset). In normal mode, it only appears as a serial port (/dev/cu.usbmodem*).

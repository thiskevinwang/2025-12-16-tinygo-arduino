# Description

This is a TinyGo project for a XIAO BLE nRF52840 board. 
- [wiki.seeedstudio.com](https://wiki.seeedstudio.com/XIAO_BLE/)
- [Amazon](amazon.com/dp/B0DRNTBPNT?ref_=ppx_hzsearch_conn_dt_b_fed_asin_title_1)

## Prerequisites

- [`tinygo`](https://tinygo.org/getting-started/install/macos/) >= 0.40.0
- XIAO BLE board. Also see [Tinygo reference](https://tinygo.org/docs/reference/microcontrollers/machine/xiao-ble/)
- USB-C cable

## Build and Flash

To flash the `main.go` program in this project, run the following from the root of this project.

```bash
tinygo flash -target=xiao-ble .
```

To flash and monitor the serial output, run:

```bash
tinygo flash -monitor -target=xiao-ble . 
```

To flash, monitor, and print stack traces on panic, run:

```bash
tinygo flash -monitor -print-stacks -target=xiao-ble .
```

To build the firmware without flashing, run:

```bash
tinygo build -target=xiao-ble .
```

## Troubleshooting

> My XIAO is connected to my mac. How do I find it?

```console
user@~: $ ls /Volumes/ && ls -l /dev/cu.* 2>/dev/null || echo "No serial ports found"

Macintosh HD
crw-rw-rw-  1 root  wheel  0x9000005 Dec 23 09:56 /dev/cu.Bluetooth-Incoming-Port
crw-rw-rw-  1 root  wheel  0x9000001 Dec 23 09:55 /dev/cu.debug-console
```

> [!NOTE]
> The XIAO BLE only shows up as a mounted volume (in Volumes) when it's in bootloader mode (double-tap reset). In normal mode, it only appears as a serial port (/dev/cu.usbmodem*).

> Flashing to specific port

```
tinygo flash -target=xiao-ble -port=/dev/cu.usbmodem101 .
```
# Hardware

The controller PCB has two processors: an ESP8266 and an STM32F407VG.

* The ESP8266 is responsible for communication with the outside world,
  principally the WiFi interface.
* The STM32 is responsible for control of the fans, pump, valve, water tank
  sensor, lighting, door, etc.

Among the many other components, there are two of particular interest for our
purposes:

* A [TS3A24157](https://www.ti.com/lit/gpn/ts3a24157). This is an SPDT switch.
  In its "normal" state, it connects the UART in/outputs of the ESP8266 with one
  of the UARTs on the STM32. If pin 1 of the `Service interface` is pulled high
  (to VCC), it instead connects pins 2 and 3 of the `Service interface` with the
  ESP8266.
* An [ATECC508A](https://www.microchip.com/en-us/product/ATECC508A). This is a
  cryptographic processor which stores the cryptographic key of the Plantcube
  and which can sign data using that key.

Both processors can be connected to with a debug connector. The connections both
use [TC2050-IDC](https://shop.collion.de/tc2050/TC2050-IDC.html) cables.

The STM32 is easy - it's connected to the `jtag` connection, which has a
standard
[ARM Cortex Pinout](https://developer.arm.com/documentation/101416/0100/Hardware-Description/Target-Interfaces/Cortex-Debug--10-pin-).
It can be controlled using e.g. an STLink V2 adapter (with the adapter providing
3.3V power).

The ESP8266 is somewhat more challenging. The pinout of the `Service interface`
is:

| Pin         | Function         | Function    | Pin          |
|-------------|------------------|-------------|--------------|
| Top left    |                  |             | Top right    |
| 5           | ESP8266 GPIO0    | GND         | 6            |
| 4           | ESP8266 RST      | ESP8266 IO2 | 7            |
| 3           | ESP8266 RX       | VCC         | 8            |
| 2           | ESP8266 TX       | N/C         | 9            |
| 1           | TS3A24157 switch | N/C         | 10           |
| Bottom left |                  |             | Bottom right |

Pin 1 should be connected to VCC in order to connect pins 2 and 3 with the
ESP8266.

The PCB has a pull-up on pin 4 (ESP8266 RST).

With a programmer such as an ESPProg, connect pin 6 to GND, pin 1 to VCC, pin 8
to VCC and pins 2 and 3 to RX/TX and communication should be possible.

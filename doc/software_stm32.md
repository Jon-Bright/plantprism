# STM32 software

## Libraries

The STM32 uses (at least) the following libraries:

* [QP real-time embedded frameworks](https://www.state-machine.com/), version
  `6.7.0`, with the `QV` kernel. It shows build date 03/11/22 and time
  09:12:08. The QP software has a number of configurable options which are
  configured as follows:
  * `Q_SIGNAL_SIZE`: 2
  * `QF_EVENT_SIZ_SIZE`: 2
  * `QF_TIMEEVT_CTR_SIZE`: 4
  * `QF_EQUEUE_CTR_SIZE`: 1
  * `QF_MPOOL_SIZ_SIZE`: 2
  * `QF_MPOOL_CTR_SIZE`: 2
  * `QS_OBJ_PTR_SIZE`: 4
  * `QS_FUN_PTR_SIZE`: 4
  * `QS_TIME_SIZE`: 4
  * `QF_MAX_ACTIVE`: 32
  * `QF_MAX_TICK_RATE`: 2
  * `QF_MAX_EPOOL`: 3
* [A modified newlib](https://github.com/lupyuen/newlib/tree/master) as C
  library.

## Modules

Various message-handling modules are created:

* `RS485Arbiter`. Controls the RS485 bus used to communicate with the LEDs.
* `DoorManager`. Deals with the door open/close sensor.
* `TempHumManager`. 
* `LEDManager`. Communicates with the LEDs.
* `FanManager`
* `CoolingManager`
* `PumpManager`
* `TankManager`. Deals with the tank water level sensor. (There's a magnet in
  the water tank's float and hall effect sensors in the Plantcube's wall near
  where it floats.)
* `ValveManager`
* `ECManager`. Unclear what this is for. It reads an ADC value.
* `TempController`
* `WaterController`
* `NetworkControllerSend`. Sends messages to the ESP8266.
* `NetworkControllerReceive`. Receives messages from the ESP8266.
* `LightController`
* `RecipeService`. Deals with recipe files (see below).

## Recipes

These were the reason for reverse-engineering the STM32 code in the first place,
as they were the only non-obvious MQTT message to the ESP8266. Many Bothans died
to bring us this information. Seriously, it took many hours of reverse
engineering to work out the format and content of the recipe file, because it
touches many parts of the STM32 code and its format is... not conducive to easy
reverse engineering.

The file starts with a header:

| Bytes | Content                                                  |
|-------|----------------------------------------------------------|
| 00-03 | Timestamp / Recipe ID. Little-endian Unix timestamp.     |
| 04-07 | `cycle_start_time`. Little-endian Unix timestamp.        |
| 08    | Layer count minus 1. In all recipes seen, this was 2.    |
| 09    | Recipe version. Must be 0x07.                            |
| 0a... | `num_blocks` for each layer in byte 08. Normally 0a-0c.  |

`num_blocks` in recipes seen has been `02` for one of the layers, `01` for
another and `00` for the third. Which of the first two was `02` and which was
`01` varied.

The header is followed by `sum(num_blocks)` two-byte blocks (so, 3 of them for
the recipes seen):

| Bytes | Content                                             |
|-------|-----------------------------------------------------|
| 00    | `num_periods`. The number of periods in this block. |
| 01    | `repetition_limit`. See "Finding the period" below. |

This is followed by a series of `period`s, each of which is 14 bytes long:

| Bytes | Content                                                            |
|-------|--------------------------------------------------------------------|
| 00-03 | `period_length` in seconds. Little-endian 32-bit integer.          |
| 04-07 | Light channel values (different colours). Four * one byte, 0-100.  |
| 08-09 | Temperature target. Little-endian 16-bit integer, degrees C * 100. |
| 0a-0b | Water target. Little-endian 16-bit integer.                        |
| 0c-0d | Watering time offset in seconds. Little-endian 16-bit integer.     |

### Finding the period

This code is odd. To find the currently active period, the algorithm seems to
be:

1. Subtract the header's `cycle_start_time` from the current Unix time _plus_
   the `total_offset` (see the [MQTT docs](mqtt.md#total_offset)). The cycle
   start time can be up to 6 months ago, so this potentially leaves a remainder
   of several million seconds. Let's call that `remainder`.
2. `repetition_count = 0`. Also, set current period to be the first period of
   the first block.
3. If `remainder` is less than 0, finish, we found the period. (This very likely
   won't be true on the first iteration.)
4. `remainder = remainder - period_length`, for the current period.
5. Move to the next period of the current block. If this was the last period of
   the current block, `repetition_count = repetition_count + 1`.
6. If `repetition_count == repetition_limit` for the current block, make the
   current period be the first period of the next block. If this was the last
   block, loop back to the first block.
   
The recipes seen have one block with a single period of 86400s (one day) and a
`repetition_limit` that varies over time, as the distance from
`cycle_start_time` increases (i.e. the recipe received from AWS was unchanged
other than an increased `repetition_limit`).  The recipes then had a second
block with two periods, one of 30600s (8.5h) and one of 55800s (15.5h). The
algorithm above would therefore result in subtracting a bunch of whole days
(the `repetition_limit` of the first block) before attempting to find the actual
current period by looping through the second block.

### Recipe values seen

For seen recipes, for the sleep period:

* Light channel values are set to `0x00` for all four channels.
* Temperature target is set to 20.00C.
* Water target is set to 0.
* Watering time is set to 28800s, or -1s.

For seen recipes, for the daytime period:

* Light channel values are set to `0x3d`, `0x27`, `0x21` and `0x0a`.
* Temperature target is set to 23.00C.
* Water target is set to 70.
* Watering time is set to 28800s or 28803s.

## Messages from ESP8266

The ESP8266 and STM32 pass messages back and forth to one another. The following
are all permitted messages from the ESP to the STM:

```
0x02 	Init, respond with CONNECTED
0x04	Ack
0x08	Recipe
0x09	Ignored? No code found that processes this message.
0x0A	Set Unix time
0x0B	Something with subcommands to change mode, pause drain tank, continue drain tank. Cleaning?
0x0C	MCU update (firmware?)
0x0E	Recipe adjust
0x10	NCU update (doesn't look like this gets handled)
0x12	Restart
0x13	Set debug level
0x14	Factory reset
0x17	Cancel water down time?
0x18	Manual watering layer A
0x19	Manual watering layer B
0x1A	Network connect?
```


## Signals

QP uses signals for communication between modules. The signals listed below are
used. Those not listed don't appear to be used. Those listed with "???" are used
but their name/function hasn't yet been discovered.

```
0x01	Q_ENTRY_SIG
0x02	Q_EXIT_SIG
0x03	Q_INIT_SIG
0x04	Q_USER_SIG
0x05	MODE_CHANGE_SIG
0x06	VERBOSE_REPORTING_SIG
0x07	GRACEFUL_SHUTDOWN_SIG
0x08	DOOR_OPEN_SIG
0x09	DOOR_CLOSE_SIG
0x0a	NETWORK_CONNECT_SIG
0x0c	MANAGER_ERROR_SIG
0x0d	???

0x0f	MANAGER_SHUTDOWN_SIG
0x10	MANAGER_REFRESH_SIG
0x11	I2C_TX_DONE_SIG
0x12	I2C_RX_DONE_SIG
0x13	USART_RECEIVED_BYTE_SIG
0x14	USART_WRITE_BYTE_SIG
0x15	ONOFF_SET_SIG
0x16	ONOFF_CHANGE_SIG
0x17	DEVICE_ON_SIG
0x18	DEVICE_OFF_SIG
0x19	GPIO_IT_CHANGE_SIG
0x1a	SAMPLE_PIN_SIG
0x1b	CONTROLLER_MODE_SIG
0x1c	DEBUG_MODE_ON_SIG
0x1d	DEBUG_MODE_OFF_SIG
0x1e	SERVICE_ERROR_SIG
0x1f	LAYER_A_SIG
0x20	LAYER_B_SIG
0x21	APPLIANCE_SIG
0x22	SOFTWARE_ERROR_SIG
0x23	DOOR_API_SIG
0x24	DEBUG_MESSAGE_SIG
0x25	TEMP_CHANGE_SIG
0x26	HUM_CHANGE_SIG
0x27	TEMPHUM_TIMEOUT_SIG
0x28	FAN_SET_SIG
0x29	FAN_BOOST_SIG
0x2a	FAN_TACHO_SIG
0x2b	FAN_RAW_RESULT_SIG
0x2c	FAN_GET_SPEED_SIG
0x2d	FAN_TACHO_ERROR_SIG
0x2e	FAN_STOP_SIG
0x2f	FAN_START_SIG
0x30	FAN_RAMP_SPEED_SIG
0x31	TANKLVL_CHANGE_SIG
0x32	TANKLVL_RAW_CHANGE_SIG
0x33	TANKLVL_AVAILABLE_SIG
0x34	EC_CHANGE_SIG
0x35	EC_SETUP_DONE_SIG
0x36	EC_TIMEOUT_SIG
0x37	EC_RAW_RESULT_SIG
0x38	EC_REPORT_SIG
0x39	RFID_CHANGE_SIG
0x3a	VALVE_SET_SIG
0x3b	VALVE_OPENED_SIG
0x3c	VALVE_CLOSED_SIG

0x3e	VALVE_CHANGE_SIG
0x3f	LED_SET_SIG
0x40	LED_TRANSFER_READY_SIG
0x41	LED_BUS_ERROR_SIG
0x42	LED_BOOT_SIG
0x43	LED_REQUEST_BUS_SIG
0x44	LED_TRANSFER_DONE_SIG
0x45	LED_BAUDRATE_SIG
0x46	LED_ADDRESS_SIG
0x47	LED_EXT_RESET_SIG
0x48	LED_RESET_CONFIG_SIG
0x49	LED_POLL_SIG
0x4a	LED_BOARD_STATUS_SIG
0x4b	RS485_BUS_GRANTED_SIG
0x4c	RS485_LED_SWITCH_SIG
0x4d	WATER_TARGET_SIG
0x4e	DRAIN_TANK_MODE_ON_SIG
0x4f	DRAIN_TANK_MODE_OFF_SIG
0x50	DRAIN_TANK_TIMEOUT_SIG
0x51	CANCEL_WATER_DOWN_TIME_SIG
0x52	MANUAL_WATERING_SIG
0x53	PAUSE_DRAIN_TANK_SIG
0x54	CONTINUE_DRAIN_TANK_SIG
0x55	SILENT_MODE_WATERING_SIG
0x56	ABORT_WATER_SIG
0x57	WATER_CYCLE_SIG
0x58	CHECK_QUEUE_SIG
0x59	SENSOR_GAP_TIMEOUT_SIG
0x5a	TANK_EMPTY_SIG
0x5b	TANK_REFILL_SIG
0x5c

0x5e	AFTER_WATERING_2H_SIG
0x5f	AFTER_WATERING_3H_SIG
0x60	TEMP_TARGET_SIG
0x61	COMPRESSOR_ECO_SIG
0x62	COMPRESSOR_UNLOCK_SIG
0x63	COMPRESSOR_MAX_SIG
0x64	MAX_RECIPE_TEMP_SIG
0x65	REPORT_ECO_WARNING_SIG
0x66	UNKNOWN_66_SIG
0x67	LIGHT_TARGET_SIG
0x68	DOOR_UPDATE_LED_SIG
0x69	AFTERCOOLING_DONE_SIG
0x6a	NETCTRL_MCU_ACK_SIG
0x6b	NETCTRL_NCU_ACK_SIG
0x6c	NETCTRL_TIMEOUT_SIG
0x6d	NETCTRL_FEED_IWDG_SIG
0x6e	NETCTRL_CONNECTED_SIG
0x6f	NETCTRL_NCU_UPDATE_SIG
0x70	NETCTRL_MCU_UPDATE_SIG
0x71	NETCTRL_YMODEM_RECEIVE_SIG
0x72	NETCTRL_YMODEM_ERROR_SIG

0x74	NETCTRL_YMODEM_ACK_SIG
0x75	NETCTRL_YMODEM_NAK_SIG
0x76	NETCTRL_YMODEM_C_SIG
0x77	NETCTRL_YMODEM_ACK_C_SIG
0x78	NETCTRL_YMODEM_END_SIG
0x79	NETCTRL_YMODEM_START_SIG
0x7a	NETCTRL_YMODEM_SENDER_ABORT_SIG
0x7b	NETCTRL_YMODEM_RECEIVER_ABORT_SIG
0x7c	NETCTRL_YMODEM_SUCCESS_SIG
0x7d	NETCTRL_YMODEM_READY_SIG
0x7e	NETCTRL_YMODEM_KILL_SESSION_SIG
0x7f	RECIPE_CHANGE_SIG
0x80	RECIPE_START_SIG
0x81	RECIPE_ADJUST_SIG
0x82	RECIPE_CONFIRMATION_SIG
0x83	RECIPE_NEW_PERIOD_SIG
0x84	SUN_SET_RISE_START?
0x85	SUN_SET_RISE_FINISHED?
0x86	SUN_SET_RISE_SIG
0x87	SILENT_MODE_END_SIG
```

## RTC

The RTC module of the STM32, in addition to storing the date and time, offers 20
bytes of battery-backed storage for user data. The Plantcube uses these bytes as
follows:

```
0x00 Set to 0xdecea5ed - this is used as a flag value for validity of the RTC data.
0x01 Recipe time offset (must be <86400).
0x02 Assertion timestamp.
0x03 Assertion line number.
0x04-0A Assertion module.
0x0B ?
0x0C ?
0x0D ?
0x0E ?
0x0F ?
0x10 ?
0x11 ?
0x12 Debug level
0x13 ?
```

The assertion data is stored by the Plantcube's `Q_onAssert` function when one
of the code's assertions fails.

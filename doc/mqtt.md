# MQTT

## Subscribed topics

On connection, the Plantcube subscribes to the following topics. `<DEVICE ID>`
in all cases is the Plantcube's UUID.

* `$aws/things/<DEVICE ID>/shadow/update/accepted`
* `$aws/things/<DEVICE ID>/shadow/update/rejected`
* `$aws/things/<DEVICE ID>/shadow/update/delta`
* `agl/prod/things/<DEVICE ID>/firmware`
* `agl/all/things/<DEVICE ID>/rpc/put`
* `agl/all/things/<DEVICE ID>/shadow/get/accepted`
* `agl/prod/things/<DEVICE ID>/recipe`

## AWS status reporting

The Plantcube regularly reports on the status of values of its choice - possibly
every time they update. These messages all have the following format:

* MQTT message type: `publish`
* Topic: `$aws/things/<DEVICE ID>/shadow/update`
* Content: a JSON map with a `clientToken` field (whose value is the same in
  every status report) and a `state` field, itself containing a map, the only
  field in which is `reported`, itself a map, with the various reported values.

### Example content

```
{
  "clientToken":"5975bc44",
  "state":{
    "reported":{
	  "wifi_level":0,
	  "firmware_ncu":1667466618,
	  "door":false,
	  "cooling":false,
	  "total_offset":68400
	}
  }
}
```

### Reported values

The following keys have been seen in state reports:

* `cooling`, boolean. Whether the cooling pump is currently running.
* `door`, boolean. Whether the door is open.
* `firmware_ncu`, integer, Unix timestamp. Firmware version of the ESP8266.
* `humid_a`, integer, percentage. Relative humidity in the upper layer. This
  doesn't seem to ever actually get published by the device.
* `humid_b`, integer, percentage. Relative humidity in the lower layer.
* `light_a`, boolean. Whether the lights on the upper layer are on.
* `light_b`, boolean. Whether the lights on the lower layer are on.
* `recipe_id`, integer, Unix timestamp/Recipe ID or 1 (used for "tell me which
  Recipe ID to use"). The ID (found in the first four bytes of the recipe) of
  the current recipe.
* `tank_level`, integer, observed values 0,2. Tank water level but smoothed?
  Doesn't update just from watering.
* `tank_level_raw`, integer, observed values 0-2. Tank water level. Updates
  every time there's watering.
* `temp_a`, decimal, degrees C. Temperature in the upper layer.
* `temp_b`, decimal, degrees C. Temperature in the lower layer.
* `temp_tank`, decimal, degrees C. Temperature in the tank. Unclear where the
  sensor for this measurement is.
* `total_offset`, integer. Appears to be a number of seconds, meaning unclear.
* `valve`, integer, observed values:
  * `0`, water to bottom layer.
  * `1`, water to top layer.
  * `4`, water off.
* `wifi_level`, integer, observed values 0-2. WiFi reception quality?

## AGL status reporting

These are less-frequent reports. Note the slightly-different topic and lack of
`clientToken`.

* MQTT message type: `publish`
* Topic: `agl/prod/things/<DEVICE ID>/shadow/update`
* Content: a JSON map with just a `state` field, itself containing a map, the
  only field in which is `reported`, itself a map, with the various reported
  values.

### Example content

```
{
  "state":{
    "reported":{
	  "ec": 1314
	}
  }
}
```

### Reported values
* `connected`, boolean. Whether the Plantcube currently has an active MQTT
  connection.
* `ec`, integer
  * Observed values 1189-1714.
  * Evidence is strong that this is "Electrical Conductivity", which is a common
  method to measure fertiliser concentration in water.
  * Reads an ADC value on the STM32. Updates happen around the time of watering.
  * The highest values were observed when cleaning (when the tank's additionally
  full of cleaning tabs).
  * Values of ~1350 appear to be low enough to trigger an "add 15ml of nutrient"
    message.
  * After adding nutrient, values of ~1550 appear.
  * Watering happens on the upper, then the lower layer. Both layers see an EC
    update, with the second one usually being higher.
  * Temperature also seems to be a factor.

## AGL mode reporting

* MQTT message type: `publish`
* Topic: `agl/prod/things/<DEVICE ID>/mode`
* Content: a JSON map with fields `prev_mode`, `mode`, `trigger`, each of which
  are integers.

### Example content

```
{
  "prev_mode": 0,
  "mode": 5,
  "trigger": 0
}
```

### Modes

* `0`, default mode.
* `1`, debug mode (not actually observed, but it's in the code).
* `2`, end of "rinse" phase of cleaning, ready for manual cleaning.
* `3`, tank drain mode (maybe only at end of cleaning?).
* `4`, tank drain mode (maybe explicit tank drain?). Not observed.
* `5`, cleaning. Has sub-modes.
* `6`, unknown (not actually observed, but it's in the code).
* `7`, silent mode. Scales LEDs down by 50%, to reduce their heat output and
  thereby reduce cooling.
* `8`, cinema mode. Scales LEDs down by 100%.

### Triggers

* `0`, seen when starting cleaning, when activating pump mode post-cleaning,
  when activating cinema mode and when activating silent mode. Presumably
  "triggered by app".
* `1`, seen at end of cleaning (when the Plantcube switches the white light on)
  and when the pump stops itself as the tank is empty. Presumably "triggered by
  device".

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

## Status reporting

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

* `connected`, boolean. Whether the Plantcube currently has an active MQTT
  connection.
* `cooling`, boolean. Whether the cooling pump is currently running.
* `door`, boolean. Whether the door is open.
* `ec`, integer, observed values 1238-1484. This is from reading an ADC value on
  the STM32, but so far, no clue what that ADC is measuring.
* `firmware_ncu`, integer, Unix timestamp. Firmware version of the ESP8266.
* `humid_a`, integer, percentage. Relative humidity in the upper layer.
* `humid_b`, integer, percentage. Relative humidity in the lower layer.
* `light_a`, boolean. Whether the lights on the upper layer are on.
* `light_b`, boolean. Whether the lights on the lower layer are on.
* `recipe_id`, integer, Unix timestamp/Recipe ID or 1 (used for "tell me which
  Recipe ID to use"). The ID (found in the first four bytes of the recipe) of
  the current recipe.
* `tank_level_raw`, integer, observed values 1-2. Tank water level.
* `temp_a`, decimal, degrees C. Temperature in the upper layer.
* `temp_b`, decimal, degrees C. Temperature in the lower layer.
* `temp_tank`, decimal, degrees C. Temperature in the tank. Unclear where the
  sensor for this measurement is.
* `total_offset`, integer. Appears to be a number of seconds, meaning unclear.
* `valve`, integer, observed values 1,4.
* `wifi_level`, integer, observed values 0-2. WiFi reception quality?

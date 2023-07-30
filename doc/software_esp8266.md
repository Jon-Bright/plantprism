# ESP8266 software

## OS

The ESP8266 uses [Mongoose OS](https://mongoose-os.com/). Mongoose OS uses a
modular structure. The Plantcube uses the following modules:

* mongoose
* ota-common
* vfs-common
* vfs-dev-part
* vfs-fs-spiffs
* core
* i2c
* atca
* mqtt
* shadow
* aws
* wifi
* http-server
* dns-sd
* fstab
* mbedtls
* ota-http-client
* rpc-common
* rpc-service-acta
* rpc-service-config
* rpc-service-fs
* rpc-uart
* sntp

## Filesystems

There are three [SPIFFS](https://github.com/pellepl/spiffs) filesystems on the
ESP8266's flash:

* Offset 0x00008000, 256KiB
* Offset 0x00048000, 256KiB
* Offset 0x00300000, 768KiB

The first two are mirrored copies of each other (presumably for the purpose of
OTA updates). They contain Mongoose OS configuration files along with the
Plantcube's client certificate and its trusted CA certificates.

The third filesystem contains a saved `recipe.bin` file along with a copy of the
STM32 firmare (`firmware.ota`). The ESP8266 is able to update the STM32's
firmware by transferring this via YModem.

## Enabling RPCs

The RPC interface is disabled by default. I've not found a way to activate it
without opening the Plantcube and connecting to the debug interface of the
ESP8266. To do this:

1. Remove the drawers and water tank from the Plantcube.
2. Hold the door and remove the screw from the top of the upper hinge. Lift the
   door off the bottom hinge.
3. Remove the two screws attaching the upper hinge to the Plantcube and remove
   it.
4. If the Plantcube is installed in a cupboard, pull it out. Caution: it's
   _very_ heavy. You'll want at least two robustly-built people.
5. On the back of the Plantcube, remove the six screws along the back top edge
   of the device.
6. Lift the top of the device up from the back, then slide it forward and lift
   it off. The controller board is now visible.
7. Attach a TC2050-IDC cable to the `Service interface` and make connections as
   described in the [hardware](hardware.md) doc.
8. Download filesystems 1 and 2 (described below).
9. Using a hex editor, look for the string `conf8.json`. A few bytes later, this
   should appear:
   ```  
   "rpc": {
    "enable": false
   },
   ```
   Change `fals` to `true` and overwrite `e` with a space. Save the file. Repeat
   the steps for filesystem 2.
10. Upload the changed filesystems to the ESP8266.
11. While you're there, you might as well download the entire ESP8266 flash.
12. Perform steps 7 to 1 in reverse.

Commands for step 8:
```
esptool --chip esp8266 --port <port> read_flash 0xf000 0x1000 spiffs1_conf8.bin
esptool --chip esp8266 --port <port> read_flash 0x4f000 0x1000 spiffs2_conf8.bin
```

Commands for step 10:
```
esptool --chip esp8266 --port <port> write_flash 0xf000 spiffs1_conf8_edit.bin
esptool --chip esp8266 --port <port> write_flash 0x4f000 spiffs2_conf8_edit.bin
```

Commands for step 11:
```
esptool --chip esp8266 --port <port> read_flash 0x0 0x400000 plantcube_esp8266_firmware.bin
```

## RPC services

After the Plantcube is restarted, a list of the available RPC services can be
obtained:

```
curl -s -d '{"id":1,"method":"RPC.List"}' <plantcube IP>:8080/rpc |jq .
```

This produces the following output:

```
{
  "id": 1,
  "src": "<Plantcube Device ID>",
  "result": [
    "Dev.Remove",
    "Dev.Erase",
    "Dev.Write",
    "Dev.Read",
    "Dev.Create",
    "FS.Umount",
    "FS.Mount",
    "FS.Mkfs",
    "FS.Rename",
    "FS.Remove",
    "FS.Put",
    "FS.Get",
    "FS.ListExt",
    "FS.List",
    "Config.Save",
    "Config.Set",
    "Config.Get",
    "ATCA.Sign",
    "ATCA.GetPubKey",
    "ATCA.GenKey",
    "ATCA.SetKey",
    "ATCA.LockZone",
    "ATCA.SetConfig",
    "ATCA.GetConfig",
    "Sys.SetDebug",
    "Sys.GetInfo",
    "Sys.Reboot",
    "RPC.Ping",
    "RPC.Describe",
    "RPC.List"
  ]
}
```

Using `FS.Get`, `FS.Put`, `FS.Rename`, `Config.Set` and `ATCA.Sign`, all further
steps can be taken to allow monitoring or diversion of the Plantcube's
communication.

## MQTT / Shadow

The Plantcube communicates via MQTT with the
[AWS IoT Device Shadow service](https://docs.aws.amazon.com/iot/latest/developerguide/iot-device-shadows.html)

When a Seedbar is planted or harvested, or Cinema mode is turned on, etc., the
Agrilution app updates the Shadow at AWS. AWS then uses MQTT to inform the
Plantcube about the change.

The Plantcube informs AWS about the status of the device (Door open/closed,
temperature, humidity, etc.) and some of these details are retrieved and
displayed by the app.

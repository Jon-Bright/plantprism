# plantprism
Ersatz-Server-Infrastruktur für die Plantcube von Agrilution.

## Hintergrund

Der Plantcube ist einen Indoorgewächshaus mit Gießfunktion, Kühlung und
UV-Beleuchtung. Sie wurde konzipiert und hergestellt von der Firma
Agrilution. Agrilution stellt ihren Geschäftsbetrieb zum 30.06.23 ein. Die
Firma hat ihren Kunden einen Rückkaufangebot für verkauften Geräte
unterbreitet.

Wir besitzen einen Plantcube seit 2020 und sind recht zufrieden mit dem
Gerät. Wir möchten daher nicht Gebrauch machen vom Rückkaufangebot und haben
untersucht, ob ein Weiterbetrieb möglich wäre.

## Sprache / Language

Meine Muttersprache ist Englisch, aber diese Seiten sind auf Deutsch, weil
ich vermute, die meisten interessierten Leser eher Deutsch sprechen. Bitte
etwaige Grammatikfehler verzeihen.

My native language is English, but these pages are in German, because I
suspect that most interested readers are German speakers.

## Stand

Ein Weiterbetrieb der Plantcube ist ohne Agrilution möglich. Sie wird
allerdings (zumindest vorerst) etwas weniger komfortabel als bisher und
etwas technisches Können erfordern.

Mein Projektplan läuft wie folgt:

1. Plantcube-Platine ausbauen, fotografieren und Verbindungen herausfinden.
2. Code aus den Prozessoren herauslesen.
3. Konfiguration der ESP8266-Prozessor modifizeren, um ein Überwachung der
   Kommunikation der Plantcube mit dem Aussenwelt zu ermöglichen.
4. Kommunikation überwachen.
5. Serverkomponente entwickeln, um der Plantcube

Stand 19.06.23 sind Schritte 1-4 fertig. Ein genauen Zeitplan für Schritt 5
gibes es nicht, aber ist meine Hoffnung, bis Ende Juni irgendwas am Laufen
zu haben.

## Technische Beschreibung des Plantcubes

### Hardware

Die Steuerplatine hat zwei Prozessoren: ein ESP8266 und ein STM32F407VG.
Der ESP8266 ist für Kommunikation mit dem Aussenwelt zuständig - also im
wesentlichen der WLAN-Schnittstelle. Der STM32-Prozessor (der bisher kaum
untersucht wurde) ist für Lüftersteuerung, Pumpe, Ventil,
Wasser-Tank-Sensor, Lichtsteuerung, Tür usw. zuständig.

Unter vielen anderen Komponenten gibt es zwei für unsere Zwecke
interessanten:
  * Ein [TS3A24157](https://www.ti.com/lit/gpn/ts3a24157). Diese ist ein
    Schalter. Im "normalen" Zustand verbindet sie die seriellen Ein- und
    Ausgänge des ESP8266 mit eines der UARTs des STM32. Wenn man allerdings
    Pin 1 der "Service Interface" mit VCC verbindet, werden stattdessen die
    Pins 2 und 3 der "Service Interface" mit dem ESP8266 verbunden.
  * Ein [ATECC508A](https://www.microchip.com/en-us/product/ATECC508A). Dies
    ist ein kryptographische Prozessor, der den kryptographischen Schlüssel
    des Plantcubes speichert und Daten mit dieser auch signieren kann.

Beide Prozessoren sind über einen Debug-Anschluss zugänglich. Die
Debug-Anschlüsse nutzen beide
[TC2050-IDC](https://shop.collion.de/tc2050/TC2050-IDC.html) Kabeln.

Der STM32 ist einfach - er ist an dem "jtag" Anschluss angebunden, der einen
[Standard ARM-Cortex Pinout](https://developer.arm.com/documentation/101416/0100/Hardware-Description/Target-Interfaces/Cortex-Debug--10-pin-)
besitzt. Er kann problemlos mit einen STLink V2-Adaptor angesprochen werden
mit 3,3V Einspeisung.

Der ESP8266 stellt eine größerer Herausforderung dar. Der Pinout des
"service interface" Anschlusses ist:

Unten Links
1. TS3A24157 "Schalter". Mit VDD verbinden, damit Pins 2 und 3 mit dem ESP
verbunden sind
2. ESP8266 TX 
3. ESP8266 RX
4. ESP8266 RST. Die Platine hat einen Pull-Up hierfür.
5. ESP8266 GPIO0
Oben Links

Oben Rechts
6. GND
7. ESP8266 IO2
8. VDD
9. N/C
10. N/C
Unten Rechts

Sobald man Pin 1 mit VDD verbunden hat und Pins 2 und 3 mit einen passenden
Programmierer (wie z.B. ESPProg), müsste die Kommunikation möglich sein.

### Software

Der ESP8266 läuft mit [Mongoose OS](https://mongoose-os.com/).  Mongoose OS
ist modular aufgebaut. Folgende Module sind beim Plantcube aktiv:

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

#### Dateisysteme

Es gibt drei [SPIFFS-Dateisysteme](https://github.com/pellepl/spiffs) im
Flash der ESP8266:

* Offset 0x00008000, 256KiB
* Offset 0x00048000, 256KiB
* Offset 0x00300000, 768KiB

Die ersten zwei sind gespiegelte Kopien voneinander (vermutlich für OTA
Zwecken) und beinhalten Mongoose OS-Konfiguration sowie der
Client-Zertifikat der Plantcube und vertraute CA-Zertifikate.

Der dritte beinhaltet ein gespeicherte `recipe.bin` Datei sowie die
STM32-Firmware (`firmware.ota`). (Der ESP8266 ist in der Lage, dies auf dem
STM32 zu überspielen mittels YModem-Transfer.)

#### RPC Einrichtung

Die RPC-Schnittstelle ist standardmäßig deaktiviert und eine Aktivierung ist
(nach bisherigen Erkenntnis) nur mit Öffnen des Plantcubes und Anschluss am
ESP8266 möglich.

1. Schuhbladen und Wassertank aus dem Plantcube nehmen.
2. Schraube am oberen Türscharnier lösen und Tür herausheben.
3. Zwei Schrauben lösen um der oberen Scharnierhalterung abzunehmen.
4. Plantcube aus seinen Schrank nehmen (falls eingebaut). Vorsicht: er ist
   sehr schwer. Mindestens zwei kräftig gebaute Personen sind
   empfehlenswert.
5. An der Rückseite des Gerätes, die sechs Schrauben nah der Oberkante
   lösen.
6. Der Gerätedeckel an seine hintere Ende nach oben heben und anschliessend
   nach vorne schieben.  Die Platine ist jetzt sichtbar.
7. TC2050-IDC Kabel am Service-Interface anschliessen, Verbindungen wie oben
   beschrieben herstellen.
8. Dateisysteme 1 und 2 herunterladen.
9. Mit Hex-Editor Dateisystem 1 nach `conf8.json` suchen. Wenige Bytes später
   müsste
   ```  
   "rpc": {
    "enable": false
   },
   ```
   stehen. `fals` in `true` ändern und `e` mit einen Leerzeichen
   überschreiben. Speichern. Wiederholen für Dateisystem 2.
10. Neue Dateisysteme auf dem ESP8266 hochladen.
11. Während man sowieso da ist, gesamten ESP8266 Flash herunterladen.
12. Schritte 7 bis 1 rückgängig machen.


Befehle für Schritt 8:
```
esptool --chip esp8266 --port <port> read_flash 0xf000 0x1000 spiffs1_conf8.bin
esptool --chip esp8266 --port <port> read_flash 0x4f000 0x1000 spiffs2_conf8.bin
```

Befehle für Schritt 10:
```
esptool --chip esp8266 --port <port> write_flash 0xf000 spiffs1_conf8_edit.bin
esptool --chip esp8266 --port <port> write_flash 0x4f000 spiffs2_conf8_edit.bin
```

Befehl für Schritt 11:
```
esptool --chip esp8266 --port <port> read_flash 0x0 0x400000 plantcube_firmware.bin
```

#### RPC Dienste

Sobald der Plantcube wieder gestartet ist, kann man eine Liste der
verfügbaren RPC-Dienste bekommen:

```
curl -s -d '{"id":1,"method":"RPC.List"}' <plantcube IP>:8080/rpc |jq .
```

Dies bringt folgende Ausgabe:

```
{
  "id": 1,
  "src": "<Plantcube Geräte-ID>",
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

Mittels `FS.Get`, `FS.Put`, `FS.Rename`, `Config.Set` und `ATCA.Sign` kann
man alle weitere erforderliche Schritte machen, um die Kommunikation der
Plantcube zu überwachen bzw. umzuleiten. Mehr Details hierzu später.

#### MQTT / Shadow

Der Plantcube kommuniziert mittels MQTT mit dem
[AWS IoT Device Shadow service](https://docs.aws.amazon.com/iot/latest/developerguide/iot-device-shadows.html)

Den App ändert, wenn man eine Pflanze einpflanzt oder erntet, oder
Kino-Modus anschaltet, usw., der Shadow bei AWS. AWS informiert der
Plantcube über die Änderung.

Der Plantcube informiert AWS über den Zustand des Geräts (Tür auf/zu,
Temperatur, Luftfeuchtigkeit usw.).

## Kontakt

Ich würde gerne in Kontakt kommen mit anderen Plantcube-Besitzer: schreibt
jon<dies hier entfernen und mit @-Zeichen ersetzen>siliconcircus.com an.

## Rechtliches

Agrilution und Plantcube sind Schutzmarken der Agrilution Systems GmbH bzw.
ihre Rechtsnachfolger. Die Marken werden hier lediglich als
Bestimmungshinweis genutzt.

Die auf diese Seiten enthaltenen Infos dürfen laut Artikel 6 der
[European Software Directive](https://eur-lex.europa.eu/legal-content/EN/ALL/?uri=CELEX%3A32009L0024)
lediglich dafür verwendet werden, um Interoperabilität der Plantcube mit
unabhängig entwickelten Software zu gewährleisten. Eine Nutzung für andere
Zwecken (z.B. um ein konkurrienden Produkt zu entwickeln) ist untersagt.


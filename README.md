# plantprism
Replacement server infrastructure for the Plantcube from Agrilution.

## Background

The Plantcube is an indoor greenhouse with functions for watering, cooling and
lighting. It was created and manufactured by the company Agrilution. Agrilution
closed down its business as of 2023-06-30. Before closing down, Agrilution
offered to buy back Plantcubes.

We own a Plantcube since 2020 and we're very satisfied with it. We therefore
didn't want to make use of Agrilution's buy-back offer. Agrilution claimed that
it would not be possible to operate the Plantcube after they shut down, as it
was "a closed ecosystem" that could only work with their app. This is
inaccurate. It's possible to continue operating the Plantcube without
Agrilution.

## Sprache / Language

Fragen oder Kontaktaufnahmen auf Deutsch sind herzlich willkommen.

These pages were originally in German in the hope of attracting other Plantcube
owners who might be interested in continued operation. Since I've received no
contact from other Plantcube owners and since it's easier for me to write in
English, I've switched to using English.

## Status

Continued operation of the Plantcube is possible without Agrilution. It won't
(at least to start with) be as convenient or as polished as with the app and
it'll need some technical knowledge.

My project plan is as follows:

1. Remove the Plantcube controller PCB from the device. Take photos and trace
   connections.
2. Read the firmware out of the processors.
3. Modify the configuration of the ESP8266 processor to allow decrypted
   monitoring of the Plantcube's communication with the outside world.
4. Monitor that communication.
5. Reverse-engineer the firmware of the processors sufficiently to understand
   the communication.
6. Develop a replacement server component to allow continued operation of the
   Plantcube.

As of 2023-07-30, steps 1-5 are finished. I don't have a firm schedule for step
6, but I'm working on it now.

## Contact

If you (still) own a Plantcube, please contact me! Write to jon&lt;remove this
and replace with an At sign&gt;siliconcircus.com

## Further information

The following pages describe various aspects of the Plantcube's operation:

[Consumables](doc/consumables.md)
[Hardware](doc/hardware.md)
[ESP8266 software](doc/software_esp8266.md)
[STM32 software](doc/software_stm32.md)

## Legal

Agrilution and Plantcube are Trademarks of Agrilution Systems GmbH or its legal
successor(s). The marks are used here solely for the purpose of identification.

According to Article 6 of the
[European Software Directive](https://eur-lex.europa.eu/legal-content/EN/ALL/?uri=CELEX%3A32009L0024),
the information on these pages may only be used to independently develop
software that's interoperable with the Plantcube. Use of the information for
other purposes (e.g. developing a competing product) is not permitted.

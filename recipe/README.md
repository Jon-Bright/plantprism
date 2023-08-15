# Recipe code

This is decompiled code from the `RecipeService_running` function of the
Plantcube's STM32 code, along with ancillary support functions. It's the exact
code running on the Plantcube, as decompiled by
[gHidra](https://ghidra-sre.org/), with variable and function names supplied by
me (where they weren't already discernible from debug strings). (And a bunch of
`field5_0xe` where there was no obvious name - although I've got some better
ideas for names since then.)

This code specifically is the start of handling `RECIPE_START_SIG` and
interprets the layers, blocks and periods in the recipe.

It's left as it originally came from gHidra, with a couple of exceptions:

* Cleanup for uncompileable code from gHidra (e.g. incorrect field names for
  unnamed fields).
* `Recipe_get_saved_total_offset_plus_current_time` is significantly changed to
  produce realistic values, since we don't have an RTC handy.
* `printf`s have been added in useful places.

The `recipe` variable contains an actual recipe observed from MQTT.

To run:

```
./build.sh
./recipe
# ... or ...
gdb ./recipe
```

Notes:

* No effort has been made to deal with issues of byte ordering. It'll compile
  and work fine on a little-endian machine and probably break horribly on
  big-endian.
* Much of this code is not the way a human would write it, because decompilers.

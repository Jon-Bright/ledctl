# ledctl

A Go server to control LED strips, supporting LPD8806 and WS281x (WS2811, WS2812, WS2815 etc.).

The WS281x support is *heavily* based on [jgarff's C version](https://github.com/jgarff/rpi_ws281x), but in contrast to the [Go bindings](https://github.com/rpi-ws281x/rpi-ws281x-go) for that library, the support here is pure Golang and doesn't use the C library. On the other hand, it only supports output via PWM, not SPI or PCM (because I didn't need them - there's nothing fundamental preventing this).

Support is also included for controlling a power supply to the LEDs. I use an ATX power supply to power the LEDs, with one GPIO pin switching the ATX "power on" switch (via a transistor) and another pin receiving the ATX "power good" signal. Before the LEDs perform an effect, power is switched on. When an effect ends, if the result is all LEDs off, power is switched off. After power-on, the code waits for the "power good" signal before proceeding to talk to the LEDs.

## Usage

```
cd ledctl
go build
# LPD8806 - all flags other than ledchip optional, these are the defaults
./ledctl --ledchip=lpd8806 --dev=/dev/spidev0.0 --spispeed=1000000 --port=24601 --pixels=160 --order=GRB &
# WS281x - all flags optional, these are the defaults
./ledctl --ledchip=ws281x --ws281xfreq=800000 --ws281xDma=10 --port=24601 --pixels=160 --order=GRB &
echo -e 'ZIP_SET_ALL 7f0000 5.0\nQUIT' |nc localhost 24601
```

Once started, the server opens the specified port and listens for connections. It recognizes the plain text commands listed below.  There are two parameters that appear repeatedly:

*colour* is a six digit hex-encoded RGB colour (eight digit for RGBW).

LPD8806: Each channel may be at most 127.  `7f7f7f` would therefore represent white, `7f0000` would be bright red, `000001` would be the dimmest possible green.
WS281x: Each channel may be at most 255.  `ffffff` would therefore represent white (on an RGB strip), `ff0000` would be bright red, `000001` would be the dimmest possible green.

*duration* is a duration for the effect, in decimal seconds.  `1.0` is exactly one second, `2.5` is two-and-a-half seconds.

```
FADE_ALL <colour> <duration>
```

Fades all LEDs to the specified colour, over the specified duration.  Will set alternate LEDs to different colours to make slower fades than the LEDs' PWM can achieve (i.e. even though LED PWM can only do 127 or 255 steps, the fading can take an arbitrarily-larger number of steps, depending on the number of available LEDs).

Returns `OK`

```
ZIP_SET_ALL <colour> <duration>
```

Sets all LEDs to the specified colour, from start (where the controller's connected) to end, over the specified duration.

Returns `OK`

```
CYCLE <duration>
```

Cycles all LEDs (i.e. all LEDs appear to be showing the same colour at any given time) through a cycle from Red to Yellow (Red+Green) to Green to Cyan (Green+Blue) to Blue to Purple (Blue+Red) to Red and so forth.  Note that each individual transition (e.g. from R 127, G 0 to R 127, G 1) is done as a fade.  Since fades will set alternate LEDs to achieve higher fidelity than the LEDs themselves can achieve, the overall effect is that a duration of 1800 (half an hour) or 3600 (an hour) can happily be given here - the colours will impercetibly change over time.

Returns `OK`.

```
RAINBOW <duration>
```

Shows a rainbow across the LEDs - one end of the strip is red, progressing through green, blue back to red at the end of the strip.  Over the given duration, offsets the starting point of the rainbow so that it gradually moves along the strip.

Returns `OK`.

```
GET
```

Returns `0` if all LEDs are completely off, `1` otherwise.

```
COLOR
```
```
COLOUR
```

Returns the hex code to which the first pixel (closest to the controller) is set.

```
MODE [<mode>]
```

Without a parameter, returns `FADE` if a fade is running, `ZIP` if a ZIP_SET_ALL is running, `CYCLE` if a cycle is running, `KNIGHTRIDER` if a Knight Rider effect is running, `CONST` if no effect is running (i.e. all LEDs have a constant colour) or `OFF` if the LEDs were turned off with an `OFF` command.

If a `mode` parameter is supplied, returns `1` if the current mode is the given mode (using the names mentioned directly above), `0` otherwise.

```
ON
```

Resumes the most recent effect.

```
OFF
```

Fades all LEDs to black over a period of 10s.

```
KNIGHTRIDER <duration>
```

Simulates the light-strip effect from Kitt, the car in the 1980s TV series "Knight Rider".

## Disclaimer

I am not associated in any way with NBC, David Hasselhoff or the creators of Knight Rider. Especially David Hasselhoff.

## Author

* [Jon Bright](https://github.com/Jon-Bright)

## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details.

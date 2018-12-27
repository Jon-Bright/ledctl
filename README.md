# ledctl

A Go server to control SPI-controlled LEDs. Specifically, I made this to control an LPD8806 strip, but I assume that it would work with RGB 5050s too.

## Usage

```
cd ledctl
export GOPATH=$(pwd)
go build main
./main --dev=/dev/spidev0.0 --port=24601 --pixels=160 --spispeed=1000000 --order=GRB &    # All flags optional, these are the defaults
echo -e 'ZIP_SET_ALL 7f0000 5.0\nQUIT' |nc localhost 24601
```

Once started, the server opens the specified port and listens for connections. It recognizes the plain text commands listed below.  There are two parameters that appear repeatedly:

*colour* is a six digit hex-encoded RGB colour.  Each channel may be at most 127.  `7f7f7f` would therefore represent white, `7f0000` would be bright red, `000001` would be the dimmest possible green.

*duration* is a duration for the effect, in decimal seconds.  `1.0` is exactly one second, `2.5` is two-and-a-half seconds.

```
FADE_ALL <colour> <duration>
```

Fades all LEDs to the specified colour, over the specified duration.  Will set alternate LEDs to different colours to make slower fades than the LEDs' PWM can achieve (i.e. even though LED PWM can only do 127 steps, the fading can take an arbitrarily-larger number of steps, depending on the number of available LEDs).

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

Capriciously unknown command.

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

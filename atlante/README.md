# Environmental variables

To turn on experimental cache support set the `ATLANTE_USED_CACHED_IMAGES` to a true value.

```
ATLANTE_USED_CACHED_IMAGES=1
```

Generated and remote assets will only be retrieved/generated if they don't already exist in the work directory.


# Supported Template functions

* to_upper : upper case the given string

`{{ to_upper "hello world" }}`
returns
HELLO WORLD


* to_lower : lower case the given string

`{{ to_lower "HELLO WORLD" }}`
returns
hello world

* format : format the given value

`{{ format "%03v" 5 }}`
returns
005

for Date values use a date format


* now : returns the current time

* div : div the first value from the second value

`{{ div 4 2 }}`
returns
2

* mul : multiple the first value from the second

* add : add the first value from the second

* sub : sub the first value from the second

* neg : negate the value

* abs : return the absolute value of the value

* seq : generate a sequence of number from the first to the second in steps of the third

* new_toggler : create a toggler that will flip between a given set of values

* rounder3 : round the value to 3 places

* first : return the first non zero value of the given set of values

* DrawBars : draw utm grid

** asIntSlice : return the set of numbers as a slice of ints (used for DrawBars)
 
** pixel_bounds :  return the set of number as a pixelBounds (used for DrawBars)

* remote : retrieve the given remote svg and return it's workdir location

```svg

{{$partial := remote "http://view-source:https://upload.wikimedia.org/wikipedia/commons/1/1a/SVG_example_markup_grid.svg}}
<rect fill="url({{$partial}}#grid)" stroke-width="2" stroke="#000" x="0" y="0" width="250" height="250"/>
<text>{{$partial}}</text>
```



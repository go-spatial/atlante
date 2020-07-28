# bestet
A cat like program that will fill in template values.

## Install

```console

$ go get github.com/gdey/cmd/bastet

```


# Example

```console
	 $ bastet name="Gautam Dey" greeting="Hello" date="25 of July"  header.tpl body.tpl
```

where the template files look like the following:

header.tpl:
```
{{greeting}} {{.name}},

```

body.tpl:
```
Thank you for comming to our event on {{.date}}, {{.name}}.
We hope you had fun, and if you have any questions please feel
free to reach out to us.

Thank you.
```


This would generate the following:

```
Hello Gautam Dey,
	
Thank you for comming to our event on 25 of July, Gautam Dey.
We hope you had fun, and if you have any questions please feel
free to reach out to us.

Thank you.
```

# Name

[Bastet](https://en.wikipedia.org/wiki/Bastet) is a cat god from ancient Egyptian religion.


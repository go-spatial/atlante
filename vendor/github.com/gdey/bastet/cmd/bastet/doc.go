/*

Command bastet is a cat like program that will fill in
template values.

	 $ go get github.com/gdey/cmd/bestet

Name value pairs are denoted using a `=`.

Example usage:

	 bestet name="Gautam Dey" greeting="Hello" date="25 of July"  header.tpl body.tpl

where the files look like the following

header.tpl:

	> {{.greeting}} {{.name}},
	>

body.tpl:

	> Thank you for comming to our event on {{.date}}, {{.name}}.
	> We hope you had fun, and if you have any questions please feel
	> free to reach out to us.
	>
	> Thank you.


This would generate the following:

	> Hello Gautam Dey,
	>
	> Thank you for comming to our event on 25 of July, Gautam Dey.
	> We hope you had fun, and if you have any questions please feel
	> free to reach out to us.
	>
	> Thank you.


The files are printed in the order they are given. Each file
will have the same name value pairs provided to them.
*/
package main

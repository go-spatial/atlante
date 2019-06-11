# grid5k

A grid provider that derives 5k grids from a configured 50k grid provider

```toml

[[providers]]
    name = "PostgisDB50K"
...

[[providers]]
    name = "5k"
    type = "grid5k"
    provider = "PostgisDB50k"

```

# Properties

The provider support s the following properties

* `type` (string) : [required] should be 'grid5k'
* `name` (string) : [required] the name of the provider (this will be normalized to the lowercase version)
* `provider` (string) :  [required] the name of a previously configured provider. (note: all provider names are normalized to lowercase)

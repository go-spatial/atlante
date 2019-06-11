# grid5k

A grid provided that cut's 50k grids into 5k grids

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

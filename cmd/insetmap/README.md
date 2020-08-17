# Inset Maps generates svg based inset maps

There are two command one is a simple cli that will take command line parameters and
a config file to generate an inset map.

The other is an http server that will serve up SVG for a requested MDGID.

# Config file

The config file is expected to be in TOML format.

An example config:

```TOML

address="${HTTP_ADDRESS}"
default="50k"

css_dir="static/css"
css_default="default"

[entry."50k"]
desc = """
50km insetmaps
"""
css_dir="static/css/50k"
css_default="default"

database = "${DB_CONNECT}"
scale = 1000
view_buffer = 0

main_sql = """
SELECT sheet, 'main_sheet' as class, ST_ASBinary(ST_Extent(wkb_geometry)) as geom
FROM public.tlm_50_index
WHERE mdg_id=$1
GROUP BY sheet
"""

adjoining_sql = """
SELECT sheet, 'ajoining' as class,  ST_ASBinary(ST_Extent(wkb_geometry)) as geom
FROM public.tlm_50_index
WHERE mdg_id != $1 AND wkb_geometry && !BBOX!
GROUP BY sheet
"""

[[entry."50k".layers]]
name = "oceans"
sql = """
SELECT 'ocean' as class, ST_AsBinary(ST_Makevalid(ST_Intersection(wkb_geometry, !BBOX!))) as geom
FROM oceans
WHERE wkb_geometry && !BBOX!
"""

[[entry."50k".layers]]
name = "coastlines"
sql = """
SELECT 'coastline' as class, ST_AsBinary(ST_Makevalid(ST_Intersection(wkb_geometry, !BBOX!))) as geom
FROM coastline
WHERE wkb_geometry && !BBOX!
"""

# These are the countries 
[[entry."50k".boundaries]]
sql = """
SELECT
    name,
    'country' as order,
    '' as parent,
    name as id,
    'country' as class,
    ST_ASBinary(ST_MakeValid(ST_Intersection(wkb_geometry, !BBOX!))) as geom
FROM
    countries0
WHERE
    wkb_geometry && !BBOX!
"""

boundary_sql = """
SELECT 
    adm0_left as name_l,
    adm0_right as name_r, 
    'country' as order, 
    '' as parent, 
    'boundary' as class, 
    ST_ASBinary(ST_MakeValid(ST_Intersection(wkb_geometry, !BBOX!))) as geom
FROM
    boundary_lines_0
WHERE
    wkb_geometry && !BBOX!
"""

[[entry."50k".boundaries]]
sql = """
SELECT
   sp.name as name, 
   'first' as order, 
   countries0.name as parent, 
   countries0.name || '.' || sp.name as id,
   'first-order' as class, 
   ST_ASBinary(ST_MakeValid(ST_Intersection(sp.wkb_geometry, !BBOX!))) as geom
FROM
   states_provinces as sp
JOIN
   countries0 on sp.adm0_a3 = countries0.adm0_a3
WHERE
   sp.wkb_geometry && !BBOX!
"""

boundary_sql = """
SELECT
   name_l,
   name_r, 
   'first' as order,
    adm0_name as parent,
   'boundary' as class,
    ST_ASBinary(ST_MakeValid(ST_Intersection(wkb_geometry, !BBOX!))) as geom
FROM 
   states_provinces_lines
WHERE
   wkb_geometry && !BBOX!
"""

```

<dl>
  <dt>address</dt>
  <dd>
    <p>This is the address the http server will listen on. If not provided or ":0" it will get an
    use and unused port. The system always prints the address it's listening on. This option is ignored
    by the cli.</p>
    <p>Note you can ask the system to use an Environment variable instead, by using `${ENV_NAME}` for the
    value. As shown in the example. Here the value of the Environemental Variable `HTTP_ADDRESS` will be 
    used instead.
    </p>
  </dd>
  <dt>default</dt>
  <dd>The entry that should be considered to be the default entry. It will be the first entry if it's not set.</dd>
  <dt>css_dir</dt>
  <dd>the directory to look at for css file that can be embedded into the svg.</dd>
  <dt>css_default</dt>
  <dd>If a css entry is not provided, which one to embed. If empty, css is not embedded.</dd>
  <dt>[entry."NAME"]</dt>
  <dd>
    <p>An entry is a grouping of data that makes up an inset map.</p>
  <dl>
<dt>database</dt>
<dd> The connection string for the database to use for this inset map</dd>
<dt>scale</dt>
<dd>The value to scale up the point for this map. 1000 is a resasonable value.</dd>
<dt>view_buffer</dt>
<dd>The amount to expand the viewbox by to create a buffer</dd>
<dt>main_sql</dt>
<dd>
  <p>The sql to use to get the main sheet's information provided an MDGID as the first parameter.</p>
  <p>The sql must return a single row with the following fields in this order:
  <ul>
    <li>sheet name</li>
    <li>svg class name</li>
    <li>wkb encoded geometry</li>
  </ul>
</dd>
<dt>adjoining_sql</dt>
<dd>
  <p>The sql to use to get the adjoining sheet's information provided an MDGID as the first parameter and a bounding box provided in the !BBOX! token.</p>
  <p>The sql must return the rows with following fields in this order:
  <ul>
  <li>sheet name</li>
  <li>svg class name</li>
  <li>wkb encoded geometry</li>
  </ul>
</dd>
<dt>css_dir</dt>
<dd>the directory to look at for css file that can be embedded into the svg. If not set it will use the gobal value. If the gobal is not set, embedding is turned off.</dd>
<dt>css_default</dt>
<dd>If a css entry is not provided, which one to embed. If empty/not set, it will use the gobal value.</dd>
<dt>layers</dt>
<dd>
  <p>These are the layers of the svg that go to build up the map. The order of the layers is the order in which they are render. Each layer rendering on top of the previous layer. </p>
  <dl>
    <dt>Name</dt>
    <dd>A name for the layer</dd>
    <dt>sql</dt>
    <dd>
    <p>The sql to use to get the geometries to be rendered within the bounding box provided in the !BBOX! token.</p>
    <p>The sql must return the rows with following fields in this order:
    <ul>
    <li>svg class name</li>
    <li>wkb encoded geometry</li>
    </ul>
    </dd>
  </dl>
  <p>Note: The autogenerated Main, Adjoining, and Cutline layers are always rendered afterwards.</p>
</dd>
<dt>boundaries</dt>
<dd>
<p>These are used to generate the boundaries diagram. The ordering is doing using the parent and order values. The parent values are used to associate the entry to the parent it belongs to. The Order value is used to create the printing order. These must be the following values:
</p>
<ul>
<li>country</li>
<li>first</li>
<li>second</li>
<li>third<li>
</ul>
<p>At some future time we may be able to remove the need for order and determine it using the parent.</p>

<dl>
<dt>sql</dt>
<dd><p>This is the main sql that get the names of the countries and provinces, parent values must be "" or the value of an id</p>
<dl>
fields and order they must appear
<dt>name</dt> <dd>name of the country or provence </dd>
<dt>order</dt><dd>order see above for values</dd>
<dt>parent</dt><dd>the parent country or provence id, or empty if this is a country</dd>
<dt>id</dt><dd>the reference id a child would put in it's parent field</dd>
<dt>class</dt><dd>styling class to associate with this region</dd>
<dt>geom</dt><dd>the geometry of the region must be a polygon or multipolygon</dd>
</dl>
</dd>

<dt>boundary_sql</dt>
<dd><p>SQL to get the boundaries of the region.</p>
<dl>fields and order they must appear
<dt>namel</dt> <dd>name of the country or provence on the left side of the line </dd>
<dt>namer</dt> <dd>name of the country or provence on the right side of the line</dd>
<dt>order</dt><dd>order see above for values</dd>
<dt>parent</dt><dd>the parent country or provence id from above, or empty if this is a country</dd>
<dt>class</dt><dd>styling class to associate with this region</dd>
<dt>geom</dt><dd>the geometry of the region must be a linestring or multilinestring</dd>
</dl>
</dd>


</dl>

</dd>
</dl>




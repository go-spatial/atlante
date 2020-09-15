# Webserver Endpoints

The system as the following server end-points.

1. <a id="get_sheets">`GET /sheets/`</a> used to get the currently configured sheets.</a>

Returns
```js
{
  "sheets": [
    {
      "name": "50k",
      "desc": "Gautam's 50k sheet",
      "scale": 50000,
      "styles": [
        {
          "name": "topo",
          "Description": "\tA style for topo maps\n\t"
        },
        {
          "name": "tegola_grids",
          "Description": "A simple style for gridded maps\n"
        }
      ]
    },
    {
      "name": "grg",
      "desc": "grg sheet",
      "scale": 50000,
      "styles": [
        {
          "name": "tegola_grids",
          "Description": "A simple style for gridded maps\n"
        },
        {
          "name": "topo",
          "Description": "\tA style for topo maps\n\t"
        }
      ]
    }
  ]
}
```

2. <a id="get_sheets_info_lng_lat"`GET /sheets/${sheet_name}/info/${lng}/${lat}` used to get the grid information for the lat and long values.</a>

Returns
```js
{
  "mdgid" : string,
  "sheet_number" : null | number,
  "jobs" : []{
     "mdgid" : string, // the mdgid 
     "sheet_number" : null | number, // the sheet number.
     "sheet_name" : string, // the sheet name
     "status" : {
        "status" : "requested" | "started" | "processing" | "completed" | "failed",
        "stage" : number (0-3), // which stage the job is at
        "total" : number (3),   // the total number of stages
         // description will represent different things depending on status.
         //  for requested, started, completed it will always be empty
         //  for processing it will be the item being processed
         //  for failed it will be the reason it failed
        "description" : string, 
     },
  },
  "pdf_url":  null | url, // if null, pdf has not be generated
  "last_generated" :  null | date, // last time the pdf was generated
  "last_edited" : date,  // last time the data was edited
  "series" : string,
  "geo_json" : geo_json // the geo_json of the bounding box.
  "lat" :  float // the queried lat
  "lng" : float // the queried lng
  "sheet_name": string
  "styles": [
        {
          "name": "tegola_grids",
          "Description": "A simple style for gridded maps\n"
        },
        {
          "name": "topo",
          "Description": "\tA style for topo maps\n\t"
        }
      ]
}
```

3. <a id="get_sheets_info_mdgid">`GET /sheets/${sheet_name}/info/mdgid/${mdgid-sheet_number}` used to get the grid information for the mdgid</a>

Returns
```js
{
  "mdgid" : string,
  "sheet_number" : null | number,
  "jobs" : []{
     "mdgid" : string, // the mdgid 
     "sheet_number" : null | number, // the sheet number.
     "sheet_name" : string, // the sheet name
     "status" : {
        "status" : "requested" | "started" | "processing" | "completed" | "failed",
        "stage" : number (0-3), // which stage the job is at
        "total" : number (3),   // the total number of stages
         // description will represent different things depending on status.
         //  for requested, started, completed it will always be empty
         //  for processing it will be the item being processed
         //  for failed it will be the reason it failed
        "description" : string, 
     },
  },
  "pdf_url":  null | url, // if null, pdf has not be generated
  "last_generated" :  null | date, // last time the pdf was generated
  "last_edited" : date,  // last time the data was edited
  "series" : string,
  "geo_json" : geo_json // the geo_json of the bounding box.
  "lat" :  float // the queried lat
  "lng" : float // the queried lng
  "sheet_name": string
  "styles": [
        {
          "name": "tegola_grids",
          "Description": "A simple style for gridded maps\n"
        },
        {
          "name": "topo",
          "Description": "\tA style for topo maps\n\t"
        }
      ]
}
```

4. <a id="post_sheets_mdgid">`POST /sheets/${sheet_name}/mdgid` will cause the pdf generation the job to start.</a>

Expected:

```js
{
   "mdgid" : string,
   "sheet_number" : null | number,
}
```

Returns:
```js
{
   "mdgid" : string,
   "sheet_number" : null | number,
   "job_id" : number,
   "status" : "requested" | "started" | "processing" | "completed",
}
```

4. <a id="post_sheets_bounds">`POST /sheets/${sheet_name}/bounds` will cause the pdf generation the job to start.</a>

Expected:

```js
{
   "bounds"         : []number, // this should be four number min_lng, min_lat, max_lng,max_lat
   "srid"           : number    // this the srid defaults to 4326
   "number_of_rows" : number    // the number of rows for a grid
   "number_of_cols" : number    // the number of cols for a grid
   "style_name"     : string    // the name of the style to use
   
}
```

Returns:
```js
{
   "mdgid" : string,
   "sheet_number" : null | number,
   "job_id" : number,
   "status" : "requested" | "started" | "processing" | "completed",
}
```

5. <a id="post_sheets_bounds_grid">`POST /sheets/${sheet_name}/bounds/grid` will return a GEOJSON describing the grid</a>

Expected:

```js
{
   "bounds"         : []number,      // this should be four number min_lng, min_lat, max_lng, max_lat
   "mdgid"          : string,        // MDGID to use to get the bounds (bounds takes precedent)
   "sheet_number"   : null | number, // used with MDGID
   "srid"           : number         // ignored (will always be in 4326)
   "number_of_rows" : number         // the number of rows for a grid
   "number_of_cols" : number         // the number of cols for a grid
   "style_name"     : string         // ignored
}
```

Returns:
```js
{
    "type": "FeatureCollection",
    "features": [{
        "type": "Feature",
        "geometry": {
            "type": "Point",
            "coordinates": [-117.164125, 32.73213]
        },
        "properties": {
            "name": "1"
        },
    }, {
        "type": "Feature",
        "geometry": {
            "type": "Point",
            "coordinates": [-117.164125, 32.683308]
        },
        "properties": {
            "name": "1"
        },
    }, {
        "type": "Feature",
        "geometry": {
            "type": "Point",
            "coordinates": [-117.133741, 32.73213]
        },
        "properties": {
            "name": "2"
        },
    }, {
        "type": "Feature",
        "geometry": {
            "type": "Point",
            "coordinates": [-117.133741, 32.683308]
        },
        "properties": {
            "name": "2"
        },
    }, {
        "type": "Feature",
        "geometry": {
            "type": "Point",
            "coordinates": [-117.179317, 32.7199245]
        },
        "properties": {
            "name": "B"
        },
    }, {
        "type": "Feature",
        "geometry": {
            "type": "Point",
            "coordinates": [-117.118549, 32.7199245]
        },
        "properties": {
            "name": "B"
        },
    }, {
        "type": "Feature",
        "geometry": {
            "type": "Point",
            "coordinates": [-117.179317, 32.6955135]
        },
        "properties": {
            "name": "A"
        },
    }, {
        "type": "Feature",
        "geometry": {
            "type": "Point",
            "coordinates": [-117.118549, 32.6955135]
        },
        "properties": {
            "name": "A"
        },
    }, {
        "type": "Feature",
        "geometry": {
            "type": "MultiLineString",
            "coordinates": [
                [
                    [-117.179317, 32.73213],
                    [-117.179317, 32.683308]
                ],
                [
                    [-117.148933, 32.73213],
                    [-117.148933, 32.683308]
                ],
                [
                    [-117.118549, 32.73213],
                    [-117.118549, 32.683308]
                ],
                [
                    [-117.179317, 32.73213],
                    [-117.118549, 32.73213]
                ],
                [
                    [-117.179317, 32.707719],
                    [-117.118549, 32.707719]
                ],
                [
                    [-117.179317, 32.683308],
                    [-117.118549, 32.683308]
                ]
            ]
        },
        "properties": {},
    }]
}
```


6. <a id="get_jobs">`GET /jobs` will return the latest 100 jobs</a>

Returns:

```js
  [{
     "mdgid" : string, // the mdgid 
     "sheet_number" : null | number, // the sheet number.
     "sheet_name" : string, // the sheet name
     "status" : {
        "status" : "requested" | "started" | "processing" | "completed" | "failed",
        "stage" : number (0-3), // which stage the job is at
        "total" : number (3),   // the total number of stages
         // description will represent different things depending on status.
         //  for requested, started, completed it will always be empty
         //  for processing it will be the item being processed
         //  for failed it will be the reason it failed
        "description" : string, 
        "pdf_url":  null | url, // if null or empty string, pdf has not be generated
        "last_generated" :  null | date, // last time the pdf was generated 
     },
     "style_location" : string, // the location of the style sheet
     "style_name" :  string, // the name configured for that style sheet
  }
  //...
  ]

```

7. <a id="get_jobs_status">`GET /jobs/%{job_id}/status` will return the status of the job</a>

Returns:

```js
  {
     "mdgid" : string, // the mdgid 
     "sheet_number" : null | number, // the sheet number.
     "sheet_name" : string, // the sheet name
     "status" : {
        "status" : "requested" | "started" | "processing" | "completed" | "failed",
        "stage" : number (0-3), // which stage the job is at
        "total" : number (3),   // the total number of stages
         // description will represent different things depending on status.
         //  for requested, started, completed it will always be empty
         //  for processing it will be the item being processed
         //  for failed it will be the reason it failed
        "description" : string, 
        "pdf_url":  null | url, // if null or empty string, pdf has not be generated
        "last_generated" :  null | date, // last time the pdf was generated 
     },
     "style_location" : string, // the location of the style sheet
     "style_name" :  string, // the name configured for that style sheet
  },

```

8. <a id="post_jobs_status">`POST /jobs/%{job_id}/status` post status updates for jobs</a>

Expected:

```js
{
        "status" : "requested" | "started" | "processing" | "completed" | "failed",
         // description will represent different things depending on status.
         //  for requested, started, completed it will always be empty
         //  for processing it will be the item being processed
         //  for failed it will be the reason it failed
        "description" : string, 
}
```

No content is returned unless there is an error.

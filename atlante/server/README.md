# Webserver Endpoints

The system as the following server end-points.

1. `GET /sheets/` used to get the currently configured sheets.

Returns
```js
{
   "sheets" : []{
        "name": string,
        "scale" : number, // the scale in meters -- 5k => 5000; 50k => 50000
        "description":string,
    }
}
```

2. `GET /sheets/${sheet_name}/info/${lng}/${lat}` used to get the grid information for the lat and long values.

Returns
```js
{
  "mdgid" : string,
  "sheet_number" : null | number,
  "pdf_url":  null | url, // if null, pdf has not be generated
  "last_generated" :  null | date, // last time the pdf was generated
  "last_edited" : date,  // last time the data was edited
  "series" : string,
  "geo_json" : geo_json // the geo_json of the bounding box.
  "lat" :  float // the queried lat
  "lng" : float // the queried lng
  "sheet_name": string
}
```

3. `GET /sheets/${sheet_name}/info/mdgid/${mdgid-sheet_number}` used to get the grid information for the mdgid

Returns
```js
{
  "mdgid" : string,
  "sheet_number" : null | number,
  "pdf_url":  null | url, // if null, pdf has not be generated
  "last_generated" :  null | date, // last time the pdf was generated
  "last_edited" : date,  // last time the data was edited
  "series" : string,
  "geo_json" : geo_json // the geo_json of the bounding box.
  "lat" :  float // the queried lat
  "lng" : float // the queried lng
  "sheet_name": string
}
```

4. `POST /sheets/${sheet_name}/mdgid` will cause the pdf generation the job to start.

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

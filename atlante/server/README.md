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
}
```

3. `GET /sheets/${sheet_name}/info/mdgid/${mdgid-sheet_number}` used to get the grid information for the mdgid

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

5. `GET /jobs` will return the latest 100 jobs

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
     },
  }
  //...
  ]

```
6. `GET /jobs/%{job_id}/status` will return the status of the job

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
     },
  },

```

7. `POST /jobs/%{job_id}/status` post status updates for jobs

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

No content is retuned unless there is an error.

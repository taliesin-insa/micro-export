# Micro-conversion API
API for the microservice exporting zip archive containing all images with their descriptive file. The "unreadable" images are placed in a "Unreadable" folder with their descriptive file.
The possible descriptive file format are:
* piFF

## Home Link [/export]
Simple method to test whether the Go API is running correctly

### [GET]
+ Response 200 (text/plain)  
	+ Body  
    	~~~
    	[MICRO-EXPORT] Welcome home!
    	~~~

## Export image with their piFF file [/export/piff]
This action returns a zip archive containing all images with their piFF file. 

### [POST]
This action has 3 negative responses defined:
The database service can return a 400 http error.
It will return status 401 if the permission to export is insufficient (from the database).    
It will return a status 500 if an error occurs in the Go service. This can happen in the authentication checking, the database retrieving, the images copying or in the files writing.  

+ Response 200 (application/zip)  
	+ Body  
		~~~
		BLOB (zip archive containing all images with their piFF file)
		~~~

+ Response 400 (text/plain)  
	+ Body  
		~~~
        {user-friendly error message}
        ~~~
  
  + Response 401 (text/plain)  
  	+ Body  
  		~~~
          {user-friendly error message}
          ~~~

+ Response 500 (text/plain)  
	+ Body  
		~~~
        [MICRO-CONVERSION] {user-friendly error message}
        ~~~












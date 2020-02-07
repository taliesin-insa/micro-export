# Export

Microservice of export developed with Go:
- get all PiFF files 

## Exposed REST API
**GET */export/piff***  
Get a zip file containing all associations *image/PiFF* in the database  
**Request body**: nothing  
**Returned data**: a zip file


## Commits
The title of a commit must follow this pattern : \<type>(\<scope>): \<subject>

### Type
Commits must specify their type among the following:
* **build**: changes that affect the build system or external dependencies
* **docs**: documentation only changes
* **feat**: a new feature
* **fix**: a bug fix
* **perf**: a code change that improves performance
* **refactor**: modifications of code without adding features nor bugs (rename, white-space, etc.)
* **style**: CSS, layout modifications or console prints
* **test**: tests or corrections of existing tests
* **ci**: changes to our CI configuration


### Scope
Your commits name should also precise which part of the project they concern. You can do so by naming them using the following scopes:
* PiFFExport
* API
* Configuration
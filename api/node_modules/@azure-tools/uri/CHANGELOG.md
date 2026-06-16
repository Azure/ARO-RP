# Change Log - @azure-tools/uri

This log was last generated on Fri, 19 Mar 2021 22:11:11 GMT and should not be manually modified.

## 3.1.1
Fri, 19 Mar 2021 22:11:11 GMT

### Patches

- **Fix** Issues with url.URL and local paths
- **Added** simplifyUri will strip extra / in paths. If there 2 consecutive /, replaces with a single one.

## 3.1.0
Fri, 19 Mar 2021 16:38:33 GMT

### Minor changes

- **Renamed** all function to be camelCase and deprecate PascalCase ones
- **Update** remove the use of deprecated node apis(`url parse` and `url resolve`)


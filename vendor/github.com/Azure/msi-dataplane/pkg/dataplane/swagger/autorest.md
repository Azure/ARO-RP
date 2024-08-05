### AutoRest Configuration

> see <https://aka.ms/autorest>

```yaml
go: true
track2: true
file-prefix: zz_generated_
module-version: 0.0.1
use-extension:
  "@autorest/modelerfour": "~4.27.0"
  "@autorest/go": "4.0.0-preview.63"
directive:
    - from: swagger-document
      debug: true
      where: $.paths[*].[*].responses[?(@.schema["$ref"] == "#/definitions/ErrorResponse")]
      transform: |
        $lib.log($);
        $['x-ms-error-response'] = true;
    - from: swagger-document
      debug: true
      where: $..description
      transform: $ = "" 
    - from: swagger-document
      debug: true
      where: $..summary
      transform: $ = "" 
```

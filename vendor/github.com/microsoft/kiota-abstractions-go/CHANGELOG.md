# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

### Changed

## [1.0.0] - 2023-05-04

### Changed

- GA Release.

## [0.20.0] - 2023-04-12

### Added

- Adds response headers to Api Error class

### Changed

## [0.19.1] - 2023-04-12

### Added

### Changed

- Fixes concurrent map write panics when enabling backing stores.

## [0.19.0] - 2023-03-22

### Added

- Adds base request builder class to reduce generated code duplication.

## [0.18.0] - 2023-03-20

### Added

- Adds utility functions `CopyMap` and `CopyStringMap` that returns a copy of the passed map.

## [0.17.3] - 2023-03-15

### Changed

- Fixes panic when updating in-memory slices, maps or structs .

## [0.17.2] - 2023-03-01

### Added

- Adds ResponseStatusCode field in ApiError struct.

## [0.17.1] - 2023-01-28

### Added

- Adds a type qualifier for backing store instance type to be `BackingStoreFactory`.

### Changed

## [0.17.0] - 2023-01-23

### Added

- Added support for backing store.

## [0.16.0] - 2023-01-10

### Added

- Added a method to convert abstract requests to native requests in the request adapter interface.

## [0.15.2] - 2023-01-09

### Changed

- Fix bug where empty string query parameters are added to the request.

## [0.15.1] - 2022-12-15

### Changed

- Fix bug preventing adding authentication key to header requests.

## [0.15.0] - 2022-12-15

### Added

- Added support for multi-valued request headers.

## [0.14.0] - 2022-10-28

### Changed

- Fixed a bug where request bodies collections with single elements would not serialize properly

## [0.13.0] - 2022-10-18

### Added

- Added an API key authentication provider.

## [0.12.0] - 2022-09-27

### Added

- Added tracing support through OpenTelemetry.

## [0.11.0] - 2022-09-22

### Add
- Adds generic helper methods to reduce code duplication for serializer and deserializers
- Adds `WriteAnyValue` to support serialization of objects with undetermined properties at execution time e.g maps.
- Adds `GetRawValue` to allow returning an `interface{}` from the parse-node

## [0.10.1] - 2022-09-14

### Changed

- Fix: Add getter and setter on `ResponseHandler` pointer .

## [0.10.0] - 2022-09-02

### Added

- Added support for composed types serialization.

## [0.9.1] - 2022-09-01

### Changed

- Add `ResponseHandler` to request information struct

## [0.9.0] - 2022-08-24

### Changed

- Changes RequestAdapter contract passing a `Context` object as the first parameter for SendAsync

## [0.8.2] - 2022-08-11

### Added

- Add tests to verify DateTime and DateTimeOffsets default to ISO 8601.
- Adds check to return error when the baseUrl path parameter is not set when needed.

## [0.8.1] - 2022-06-07

### Changed

- Updated yaml package version through testify dependency.

## [0.8.0] - 2022-05-26

### Added

- Adds support for enum and enum collections responses.

## [0.7.0] - 2022-05-18

### Changed

- Breaking: adds support for continuous access evaluation.

## [0.6.0] - 2022-05-16

- Added a method to set the content from a scalar value in request information.

## [0.5.0] - 2022-04-21

### Added

- Added vanity methods to request options to add headers and options to simplify code generation.

## [0.4.0] - 2022-04-19

### Changed

- Upgraded uri template library for quotes in template fix.
- Upgraded to Go 18

## [0.3.0] - 2022-04-08

### Added

- Added support for query parameters with special characters in the name.

## [0.2.0] - 2022-04-04

### Changed

- Breaking: simplifies the field deserializers.

## [0.1.0] - 2022-03-30

### Added

- Initial tagged release of the library.

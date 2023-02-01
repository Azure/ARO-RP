# Admin Portal Version 2

## REQUIRED: Install Node and NPM
> Using NVM is easiest
```
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.1/install.sh | bash

nvm install 16.16.0

nvm use 16.16.0
```

## Installing, Auditing & Building
> Below are instructions to install dependencies and run a build test.

## Install
```
cd portal/v2

npm install
```

## Audit
```
npm audit
```

> Expected output: `found 0 vulnerabilities`

## Polyfills
With the introduction of react-scripts and webpack v5.x.x polyfills for node.js core modules are no longer included by default.

As such, we need to employ a configuration override to dependencies that require them. You can find these in [config-overrides](./config-overrides.js)

After adding the required fallback in `Object.assign` you need to `npm install --save-dev` the package that is needed.

## Build
```
npm run build
```
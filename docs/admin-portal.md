# Admin Portal

The admin portal is a SRE-facing front end used for proxying access to cluster control planes and cluster Prometheus.

The admin portal also serves a static Prometheus web frontend. The contents are taken from a Prometheus release's web-ui artifact (e.g. [2.48](https://github.com/prometheus/prometheus/releases/download/v2.48.0/prometheus-web-ui-2.48.0.tar.gz)), and the static/react subdirectory is mirrored to this repository's pkg/portal/assets/prometheus-ui directory.

## Running Admin Portal in development

1. Run `make run-portal` (containerised) or `make runlocal-portal` (non-containerised)
1. Go to `https://localhost:8444` to view the admin portal running

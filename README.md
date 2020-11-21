[![Build][gh-actions-build-image]][gh-actions-build-url]
[![Docker Build][gh-actions-docker-image]][gh-actions-docker-url]
[![Total Downloads][gh-downloads-image]][gh-downloads-url]
[![hairyhenderson/dockerhub_ratelimit_exporter on DockerHub][dockerhub-image]][dockerhub-url]

# dockerhub_ratelimit_exporter

A Prometheus-format exporter to report on DockerHub per-image rate limits.

Recently, DockerHub has [introduced rate limiting](https://docs.docker.com/docker-hub/download-rate-limit/)
for anonymous and free-tier users. This exporter can help Prometheus users
track the remaining pulls for certain images.

## Status/Roadmap

This is brand-new, hacked-together over the course of a couple hours. It works,
but I'm certain there are bugs.

This is currently very _thin_ on features. Some things that are missing:

- support for authenticating to DockerHub
- support for caching auth tokens on a per-repo basis
- exposing a few more general-purpose metrics

## Usage

The image name is provided as a query parameter when querying the exporter,
and Prometheus-format metrics are returned, like this:

```console
$ curl http://localhost:9766/limits?image=hairyhenderson/gomplate:v3.8.0
# HELP dockerhub_ratelimits_limit total number of pulls that can be performed during the window
# TYPE dockerhub_ratelimits_limit gauge
dockerhub_ratelimits_limit{image="hairyhenderson/gomplate:v3.8.0"} 100
# HELP dockerhub_ratelimits_remaining number of pulls remaining for the window
# TYPE dockerhub_ratelimits_remaining gauge
dockerhub_ratelimits_remaining{image="hairyhenderson/gomplate:v3.8.0"} 100
# HELP dockerhub_ratelimits_window_seconds the length of the time window
# TYPE dockerhub_ratelimits_window_seconds gauge
dockerhub_ratelimits_window_seconds{image="hairyhenderson/gomplate:v3.8.0"} 21600
```

## Prometheus Configuration

When using Prometheus to scrape the exporter, the `image` parameter can be
provided like this:

```yaml
scrape_configs:
  - job_name: hub-limits
    scrape_interval: 15s
    metrics_path: /limits
    params:
      image:
        - busybox
        - hairyhenderson/gomplate:v3.8.0
        - prom/prometheus:v2.22.1
    static_configs:
      - targets:
        - localhost:9766
```

## License

[The MIT License](http://opensource.org/licenses/MIT)

Copyright (c) 2020 Dave Henderson

[gh-actions-build-image]: https://github.com/hairyhenderson/dockerhub_ratelimit_exporter/workflows/Build/badge.svg
[gh-actions-build-url]: https://github.com/hairyhenderson/dockerhub_ratelimit_exporter/actions?workflow=Build

[gh-actions-docker-image]: https://github.com/hairyhenderson/dockerhub_ratelimit_exporter/workflows/Docker%20Build/badge.svg
[gh-actions-docker-url]: https://github.com/hairyhenderson/dockerhub_ratelimit_exporter/actions?workflow=Docker%20Build

[gh-downloads-image]: https://img.shields.io/github/downloads/hairyhenderson/dockerhub_ratelimit_exporter/total.svg
[gh-downloads-url]: https://github.com/hairyhenderson/dockerhub_ratelimit_exporter/releases

[dockerhub-image]: https://img.shields.io/badge/docker-ready-blue.svg
[dockerhub-url]: https://hub.docker.com/r/hairyhenderson/dockerhub_ratelimit_exporter

# Recommendation engine

Recengine is a recommendation microservice that implements "true" user-based
collaborative filtering. The engine is designed to run as a microservice
with REST interface.

Expected performance in processing 1M of users with 100 likes in avg on a
machine with 1 Core i3, 300 Mb/s SSD, 200 Mb of free RAM is:
RT = ~2.7s, RPS = ~1200.
Increasing RAM should proporionally increase RPS.
Increasing SSD speed should proportionally increase RT.
Decreasing average like count or user count should proportionally increase RT.
It's assumed, that at some load point CPU becomes the bottleneck, so increasing
its speed must fix this issue for a while until the limit is reached. Then only
the sharding feature will help to scale the service horizontally.

Current status: in development.

## Prerequisites

1. [Go runtime](https://go.dev/doc/install)

## Development installation

```bash
go mod vendor
go mod tidy
```

# Huawei Cloud Log Stream

A simple service to continuosly fetch logs from Huawei Cloud LTS and output it to stdout.

## Authentication

See <https://github.com/huaweicloud/huaweicloud-sdk-go-v3#241-environment-variables-top>.

## Configuration

See [config.go](./config.go#L12-L18) for configuration item.

## Example

See [examples](examples) on how to use it in docker container combined with [vector](https://vector.dev/) to route the logs.

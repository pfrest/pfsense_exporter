# pfSense Exporter

A Prometheus exporter for scraping metrics from pfSense firewalls using the [REST API](https://github.com/jaredhendrickson13/pfsense-api) package. This exporter collects pfSense-specific metrics that are not available to the usual node-exporter on pfSense. A full list of available metrics can be found [here](docs/METRICS.md).

> [!IMPORTANT]
> This exporter requires the [REST API](https://github.com/jaredhendrickson13/pfsense-api) package to be installed on the pfSense firewall. This exporter will not work without it.

## Installing

The exporter is designed to run externally to your pfSense instances (although the FreeBSD build can run directly on pfSense). It can be installed on any system that can reach your pfSense instance(s). To install the pre-built binaries, download the latest release from the [releases page](https://github.com/pfrest/pfsense_exporter/releases).

## Configuration

Below are the configuration options available for the pfSense Exporter:

### Top-Level Options

| Option        | Type    | Default      | Description                                                                                |
|---------------|---------|--------------|--------------------------------------------------------------------------------------------|
| `address`     | string  | `localhost`  | The address the exporter will bind to. Must be a valid IP address or `localhost`.          |
| `port`        | int     | `9945`       | The port the exporter will listen on. Must be between 1 and 65535.                         |
| `targets`     | array   | —            | Configurations for pfSense targets to scrape. See [Target Options](#target-options) below. |

### Target Options

Each item in the `targets` array has the following options:

| Option                      | Type    | Default   | Description                                                                                   |
|-----------------------------|---------|-----------|-----------------------------------------------------------------------------------------------|
| `host`                      | string  | -         | Hostname or IP address of the pfSense target. **Required.**                                   |
| `port`                      | int     | -         | Port number of the pfSense target. Must be between 1 and 65535. **Required.**                 |
| `scheme`                    | string  | `https`   | URL scheme to use for the target. Must be `http` or `https`.                                  |
| `auth_method`               | string  | —         | Authentication method. Must be `basic` or `key`. **Required.**                                |
| `username`                  | string  | —         | Username for basic authentication. Required if `auth_method` is `basic`.                      |
| `password`                  | string  | —         | Password for basic authentication. Required if `auth_method` is `basic`.                      |
| `key`                       | string  | —         | API key for key-based authentication. Required if `auth_method` is `key`.                     |
| `validate_cert`             | bool    | —         | Whether to validate the TLS certificate. If false, a warning is logged.                       |
| `timeout`                   | int     | `30`      | Timeout (in seconds) for requests to the target. Must be between 5 and 360.                   |
| `collectors`                | array   | —         | List of collectors to enable for this target. If empty, all collectors are enabled.           |
| `max_collector_concurrency` | int     | `4`       | Maximum number of collectors allowed to run concurrently. Must be between 1 and 10.           |
| `max_collector_buffer_size` | int     | `100`     | Maximum size of the collector's metric buffer. Must be at least 10. Large pfSense instances may need this value increased.                           |

## Running the Exporter

To run the exporter, execute the following command:

```bash
./pfsense_exporter --config /path/to/config.yml
```

## Scraping the Exporter

Once your exporter is running, you will need to configure a job in your Prometheus server to scrape the metrics from the exporter. Here is an example configuration:

```yaml
scrape_configs:
  - job_name: 'pfsense_exporter'
    metrics_path: /metrics 

    # List the pfSense targets you want Prometheus to scrape. Each target must also be defined in your exporter configuration file!
    static_configs:
      - targets:
          - 'host1.example.com'
          - 'host2.example.com'
          - '192.168.1.50'

    relabel_configs:
      # This converts target to the '?target=' URL parameter.
      - source_labels: [__address__]
        target_label: __param_target

      # This sets the actual scrape address to be your exporter's address.
      - source_labels: [__param_target]
        target_label: __address__
        replacement: 'localhost:9945'  # <-- Your exporter's host and port

      # Optional: This sets the 'instance' label to the original target address (your pfSense host)
      - source_labels: [__param_target]
        target_label: instance
```

## Docker

The exporter can also be run as a Docker container. To pull and run the Docker image, use the following command:

```bash
docker run \
  -p 9945:9945 \
  -v /path/to/config.yml:/pfsense_exporter/config.yml \
  ghcr.io/pfrest/pfsense_exporter:latest
```

> [!IMPORTANT]
> Be sure to change the `-p` argument to match the port specified in your exporter config and the`-v` argument to the correct path for your config file.

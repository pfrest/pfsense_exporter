# pfSense Exporter Quickstart Guide

If you're short on time and want to see the pfSense Exporter in action quickly, you can use the provided Docker Compose setup to get a full Prometheus + pfSense Exporter + Grafana stack up and running. This process takes just a few minutes.

> [!WARNING]
> This Docker Compose stack should not be used in production environments. It is intended for demonstration and testing purposes only.

## Prerequisites

Before you begin, ensure you have the following prerequisites met:

- Docker and Docker Compose installed on your machine. You can download them from [Docker's official website](https://www.docker.com/get-started).
- A pfSense instance with the [REST API](https://pfrest.org/) package installed and configured.
- A code editor with the repository cloned to your local machine.

## Step 1: Modify the exporter configuration

Open [`examples/exporter.config.yml`](/examples/exporter.config.yml) in your code editor. Update the `targets` section with the details of your pfSense instance(s). Make sure to set the correct `host`, `port`, `auth_method`, and authentication credentials.

## Step 2: Modify the Prometheus configuration

Open [`examples/prometheus.config.yml`](/examples/prometheus.config.yml) in your code editor. Update the `static_configs` > `targets` section to match the `host` values you set in the exporter configuration.

> [!IMPORTANT]
> You must ensure the `targets` listed in the Prometheus configuration match the `host` values defined in the exporter configuration file exactly, otherwise Prometheus will not be able to scrape metrics from the exporter.

## Step 3: Start the Docker Compose stack

Once your updated configs are saved, you can run the following command from the root of the repository to start the Docker Compose stack:

```bash
docker-compose -f examples/docker-compose.yml up -d
```

This command will pull the necessary Docker images and start the Prometheus server, pfSense Exporter, and Grafana instance in detached mode. Once the stack is running, you can access the services as follows:

- Prometheus: `http://localhost:9090`
- pfSense Exporter: `http://localhost:9945/metrics`
- Grafana: `http://localhost:3000` (default login: `admin`/`admin`)

# pfSense Exporter Metrics Reference

This document lists all Prometheus metrics exposed by each collector in this exporter, including their names, labels, and descriptions.

---

## `carp` Collector

| Metric Name                          | Labels                                 | Description                                         |
|--------------------------------------|----------------------------------------|-----------------------------------------------------|
| `pfsense_carp_enabled`               | host                                   | Whether CARP is enabled (1 = enabled, 0 = disabled).|
| `pfsense_carp_maintenance_mode_enabled` | host                                | Whether CARP maintenance mode is enabled (1 = enabled, 0 = disabled). |
| `pfsense_carp_virtual_ip_status`     | host, carp_status, uniqid, subnet, vhid, interface | CARP virtual IP status (1 = MASTER, 0 = BACKUP, -1 = OTHER). |

---

## `firewall_state` Collector

| Metric Name                          | Labels   | Description                                         |
|--------------------------------------|----------|-----------------------------------------------------|
| `pfsense_firewall_states_maximum_count` | host   | Maximum number of firewall states allowed by the host. |
| `pfsense_firewall_states_current_count` | host   | Current number of firewall states registered on the host. |
| `pfsense_firewall_states_usage_ratio` | host   | Ratio of firewall states currently in use as a decimal (0.0-1.0). |

---

## `firewall_schedule` Collector

| Metric Name                          | Labels      | Description                                         |
|--------------------------------------|------------|-----------------------------------------------------|
| `pfsense_firewall_schedule_active`   | host, name | Whether the firewall schedule is currently active (1) or inactive (0). |

---

## `gateway` Collector

| Metric Name                      | Labels                               | Description                                         |
|----------------------------------|--------------------------------------|-----------------------------------------------------|
| `pfsense_gateway_loss_ratio`     | host, name, srcip, monitorip         | The loss ratio of the gateway as a decimal (0.0 - 1.0). |
| `pfsense_gateway_delay_seconds`  | host, name, srcip, monitorip         | The delay of the gateway in seconds.                |
| `pfsense_gateway_stddev_seconds` | host, name, srcip, monitorip         | The standard deviation of the gateway delay in seconds. |
| `pfsense_gateway_up`         | host, name, srcip, monitorip, substatus | The status of the gateway (0 = down, 1 = up).      |

---

## `interface` Collector

| Metric Name                        | Labels                             | Description                                         |
|------------------------------------|------------------------------------|-----------------------------------------------------|
| `pfsense_interface_up`             | host, name, descr, hwif, status    | Whether the interface is up (1) or down (0).        |
| `pfsense_interface_in_errs_count`  | host, name, descr, hwif            | The number of input errors on the interface.        |
| `pfsense_interface_out_errs_count` | host, name, descr, hwif            | The number of output errors on the interface.       |
| `pfsense_interface_collisions_count` | host, name, descr, hwif           | The number of collisions on the interface.          |
| `pfsense_interface_in_bytes`       | host, name, descr, hwif            | The number of input bytes on the interface.         |
| `pfsense_interface_in_pass_bytes`  | host, name, descr, hwif            | The number of input bytes passed on the interface.  |
| `pfsense_interface_out_bytes`      | host, name, descr, hwif            | The number of output bytes on the interface.        |
| `pfsense_interface_out_pass_bytes` | host, name, descr, hwif            | The number of output bytes passed on the interface. |
| `pfsense_interface_in_pkts_count`  | host, name, descr, hwif            | The number of input packets handled by the interface. |
| `pfsense_interface_in_pass_pkts_count` | host, name, descr, hwif          | The number of input packets passed on the interface. |
| `pfsense_interface_out_pkts_count` | host, name, descr, hwif            | The number of output packets handled by the interface. |
| `pfsense_interface_out_pass_pkts_count` | host, name, descr, hwif          | The number of output packets passed on the interface. |

---

## `login_protection` Collector

| Metric Name                                 | Labels     | Description                                         |
|---------------------------------------------|------------|-----------------------------------------------------|
| `pfsense_login_protection_blocked_ip`       | host, ip   | Contains details about IPs blocked by Login Protection. |
| `pfsense_login_protection_blocked_ip_count` | host       | Current number of IPs actively blocked by Login Protection. |

---

## `package` Collector

| Metric Name                        | Labels                                         | Description                                         |
|------------------------------------|------------------------------------------------|-----------------------------------------------------|
| `pfsense_package_update_available` | host, name, shortname, installed_version, latest_version | Whether an update is available for the package (1 or 0). |

---

## `restapi` Collector

| Metric Name                          | Labels                                         | Description                                         |
|--------------------------------------|------------------------------------------------|-----------------------------------------------------|
| `pfsense_restapi_update_available`   | host, current_version, latest_version, latest_version_release_date | Whether a REST API update is available (1 = available, 0 = not available). |

---

## `service` Collector

| Metric Name                | Labels     | Description                                         |
|---------------------------|------------|-----------------------------------------------------|
| `pfsense_service_up`      | host, name | Whether the service is up (1) or down (0).           |
| `pfsense_service_enabled` | host, name | Whether the service is enabled (1) or disabled (0).  |

---

## `system` Collector

| Metric Name                      | Labels   | Description                                         |
|----------------------------------|----------|-----------------------------------------------------|
| `pfsense_system_temperature_celsius` | host  | Current system temperature in Celsius.              |
| `pfsense_system_cpu_count`       | host     | Number of CPU cores available on the system.        |
| `pfsense_system_cpu_usage_ratio` | host     | Current CPU usage as a decimal (0.0 - 1.0).         |
| `pfsense_system_disk_usage_ratio` | host    | Current disk usage as a decimal (0.0 - 1.0).        |
| `pfsense_system_memory_usage_ratio` | host  | Current memory usage as a decimal (0.0 - 1.0).      |
| `pfsense_system_swap_usage_ratio` | host    | Current swap usage as a decimal (0.0 - 1.0).        |
| `pfsense_system_mbuf_usage_ratio` | host    | Current mbuf usage as a decimal (0.0 - 1.0).        |
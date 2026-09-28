# Mission: Deploy dmama_api with NSSM

## Why
Deploy `dmama_api` on Windows Server 2019 as a reliable Windows service that starts automatically, loads the correct production configuration, and can be verified or rolled back safely.

## Success looks like
- Build and package the Go API for Windows x64.
- Install and operate it as the `dmama_api` NSSM service.
- Verify health, logs, database readiness, restart behavior, and firewall exposure.
- Upgrade or roll back the binary with a short, repeatable procedure.

## Constraints
- Windows Server 2019 and PowerShell run as Administrator.
- Secrets stay outside Git in the deployment `.env` file.
- Existing database, MongoDB, reverse-proxy, and security policies must be preserved.

## Out of scope
- Provisioning PostgreSQL, MongoDB, DNS, TLS certificates, or a load balancer from scratch.

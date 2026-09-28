# Windows NSSM Deployment Resources

## Knowledge

- [NSSM usage](https://www.nssm.cc/usage)
  Official documentation for installing a service, setting its application directory, log redirection, environment, restart behavior, and log rotation.
- [NSSM command-line reference](https://www.nssm.cc/commands)
  Official syntax for `install`, `set`, service accounts, and startup modes.
- [Service accounts in Windows Server](https://learn.microsoft.com/en-us/windows-server/identity/ad-ds/manage/understand-service-accounts)
  Microsoft guidance for choosing service identities, including passwordless virtual accounts supported by Windows Server 2019.
- [New-NetFirewallRule](https://learn.microsoft.com/en-us/powershell/module/netsecurity/new-netfirewallrule)
  Microsoft reference for adding a restricted inbound TCP rule when direct access to the API port is required.

## Wisdom (Communities)

- The organisation's Windows Server, database, and network administrators
  Confirm the service identity, database routes, firewall source subnet, backup policy, and reverse-proxy ownership before production rollout.

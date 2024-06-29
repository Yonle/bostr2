# bostr2
[bostr](https://github.com/Yonle/bostr) next generation

# Installation instructions
You will need to install [Go](https://go.dev) on your system. After installation, run the following command:
```
go install github.com/Yonle/bostr2@latest
```

Then create a YAML configuration file that looks like this:
```yaml
---
listen: localhost:8080

relays:
- wss://relay.example1.com
- wss://relay.example2.com

nip_11:
  name: bostr2
  description: Noetr relay bouncer
  software: git+https://github.com/Yonle/bostr2
  pubkey: 0000000000000000000000000000000000000000000000000000000000000000
  contact: unset
  supported_nips:
  - 1
  - 2
  - 9
  - 11
  - 12
  - 15
  - 16
  - 20
  - 22
  - 33
  - 40
```

Save the configuration file as config.yaml. Once completed, you can run bostr2 using the following command:
```
bostr2
```

You will then need to manually configure a reverse proxy on your server (eg [nginx](https://nginx.org) or [caddy](https://caddyserver.com)).

# СУКА БЛЯТЬ (blyat)
[bostr](https://github.com/Yonle/bostr) следующего поколения

# Инструкция по установке
Вам понадобится установить [Go](https://go.dev) в вашей системе. После установки выполните следующую команду:
```
go install github.com/Yonle/blyat@latest
```

Затем создайте файл конфигурации YAML, который выглядел бы следующим образом:
```yaml
---
listen: localhost:8080

relays:
- wss://relay.example1.com
- wss://relay.example2.com

nip_11:
  name: blyat
  description: Your relays, OUR RELAYS
  software: git+https://github.com/Yonle/blyat
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

Сохраните файл конфигурации как config.yaml. После завершения вы можете запустить blyat с помощью следующей команды:
```
blyat
```

Затем вам нужно будет вручную настроить обратный прокси-сервер на вашем сервере (например [nginx](https://nginx.org) или [caddy](https://caddyserver.com)).

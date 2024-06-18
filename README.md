# СУКА БЛЯТЬ (blyat)
[bostr](https://github.com/Yonle/bostr) следующего поколения

Хотя основная часть уже заработала, на самом деле дела обстоят не очень хорошо.

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
- wss://nos.lol
- wss://relay.damus.io

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
  - 42
  - 50
```

Сохраните файл конфигурации как config.yaml. После завершения вы можете запустить blyat с помощью следующей команды:
```
blyat
```

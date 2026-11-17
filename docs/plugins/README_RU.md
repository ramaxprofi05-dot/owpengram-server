# Плагины FemboyGram — первый контракт

Ветка femboygram содержит первый серверный слой plugin system: строгую схему manifest, проверку версии, target, entrypoint, permissions, checksum/signature metadata и защиту путей архива от zip-slip.

Это **ещё не готовый runtime или магазин**. Код плагинов пока не исполняется, пакеты не устанавливаются и подписи ещё не проверяются доверенным key registry. Эти шаги должны быть реализованы до включения каталога.

## Манифест

Обязательные поля:
- id — reverse-domain ID, например dev.example.weather;
- name, author, description;
- version — SemVer;
- target — android или server;
- entrypoint — относительный путь внутри пакета;
- permissions — только разрешения из allowlist для соответствующего target.

Для публикации/установки дополнительно обязательны sha256, signer_key_id и Ed25519 signature. Парсер проверяет формат подписи; будущий installer обязан криптографически проверить подпись по доверенному публичному ключу и сверить SHA-256 фактического архива.

## Разрешения

Android:
- ui:chat-list, ui:chat-screen, ui:profile;
- events:message-metadata, events:profile-public;
- storage:plugin;
- network:https.

Server:
- server:message-metadata-hook, server:profile-public-hook;
- storage:plugin;
- network:https.

Разрешения на полный файловый доступ, токены, ключи сессий, секреты приложения, native code и административные операции не входят в allowlist. Плагины не должны получать прямой доступ к базе данных или MTProto auth keys.

## Следующие обязательные этапы
1. Реестр доверенных Ed25519 publisher keys и проверка подписи архива.
2. ZIP/package importer с лимитами размера, количеством файлов, запретом symlink и проверкой каждого пути.
3. Версионированный Plugin API и capability broker.
4. Server runtime в WASM sandbox с лимитами памяти/CPU и сетевым allowlist.
5. Android runtime через ограниченный IPC/JS bridge; никакого произвольного загрузчика DEX/JAR.
6. Каталог, просмотр разрешений до установки, отключение, удаление, rollback и audit log.

До завершения этих этапов устанавливать сторонний исполняемый код нельзя.
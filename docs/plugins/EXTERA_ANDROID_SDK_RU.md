# Extera Android Plugin SDK — draft contract v1

Этот документ фиксирует контракт расширений для Extera поверх существующего
Android-клиента OwpenGram/FemboyGram. Android-клиент теперь содержит строгий
парсер манифеста (`PluginManifest`), allowlist (`PluginCapability`) и модель
проверки изменений разрешений (`PluginPermissionReview`). Они проверяют структуру
и совместимость, но не запускают код и не подтверждают доверие к подписи.
Runtime остаётся выключенным до реализации песочницы, проверки подписи на
клиенте и capability broker.

## Пакет

Пакет — ZIP с `manifest.json`, JS entrypoint (`.js` или `.mjs`) и ресурсами.
Максимальный размер, число файлов, digest и подпись проверяются серверным
валидатором `internal/plugins`. Путь каждого файла должен быть относительным;
архивы с traversal, symlink, повторяющимися путями и неизвестными полями
манифеста отклоняются.

Пример манифеста:

```json
{
  "id": "dev.example.profile-tools",
  "name": "Profile Tools",
  "version": "1.0.0",
  "author": "Example Publisher",
  "description": "Adds optional profile presentation tools.",
  "target": "android",
  "entrypoint": "dist/main.js",
  "permissions": ["ui:profile", "ui:profile-banner"]
}
```

Installable packages additionally require `sha256`, `signer_key_id`, and an
Ed25519 `signature`. The package digest covers every file except
`manifest.json`; the signature covers the canonical manifest payload.

## Initial extension points

| Capability | Scope | Contract |
| --- | --- | --- |
| `ui:chat-list` | Chat list | Add bounded, declarative list decorations |
| `ui:chat-screen` | Chat screen | Add isolated, non-blocking UI panels |
| `ui:profile` | Public profile | Add profile presentation modules |
| `ui:profile-banner` | Profile banner | Customize safe banner overlays/metadata only |
| `events:message-metadata` | Message metadata | Receive minimized, non-content metadata events |
| `events:profile-public` | Public profile | Receive public profile lifecycle events |
| `storage:plugin` | Plugin namespace | Read/write only this plugin's quota-limited storage |
| `network:https` | Network | HTTPS only through broker and user-visible policy |

Capabilities are opt-in. An absent capability means the API is unavailable;
unknown capabilities must fail closed. A plugin must not receive message text,
media bytes, contacts, auth keys, tokens, or private profile fields through
these presentation hooks.

## Runtime boundary

- Plugins execute outside the main app process in a constrained runtime.
- The host exposes a versioned bridge; plugins never call Android internals.
- UI output is declarative and size-limited; arbitrary native views are not
  accepted from packages.
- Each invocation has a deadline and resource budget. A timeout or crash
  disables that plugin instance without taking down the host app.
- Network requests pass through a broker with HTTPS enforcement, quotas, and
  clear permission state.
- Plugin storage is namespaced, quota-limited, and removed on uninstall unless
  the user explicitly chooses an export.
- No DEX/JAR loading, reflection bridge, shell, unrestricted filesystem,
  hidden API access, or direct database/MTProto access.

## Реализованная Android-основа

- `PluginManifest.parse(byte[])` ограничивает размер, отклоняет неизвестные
  поля, проверяет обязательные строки, SemVer, безопасный entrypoint и
  совместимость permissions с target.
- `PluginCapability` содержит единый клиентский allowlist capability.
- `PluginPermissionReview.compare(...)` вычисляет новые и удалённые разрешения;
  новые требуют отдельного согласия, удалённые должны быть отозваны.
- Instrumentation-тесты покрывают валидный манифест, неизвестные поля и
  capability, небезопасные пути и обновление разрешений.

Парсер не является проверкой подписи: `isInstallable()` лишь проверяет наличие
корректно сформированных полей целостности. Установка должна быть заблокирована,
пока доверенный verifier не проверит digest архива и Ed25519-подпись.

## Permission lifecycle

1. Show publisher, package digest, target, version, and requested capabilities.
2. Explain each capability in user-facing language before installation.
3. Grant only capabilities explicitly accepted by the user.
4. On update, compare requested capabilities with the installed version.
5. If a new capability appears, pause the update for renewed consent.
6. Allow disable, revoke, uninstall, and rollback without losing chats or
   account data.

## Profile banner extension behavior

`ui:profile-banner` may decorate the banner surface with bounded overlays and
public metadata. It may not replace the underlying media source, change the
owner's privacy settings, start audio, or access the original video bytes.
The host owns playback, mute state, lifecycle, reduced-motion behavior, and
fallback rendering. A plugin that fails to render is omitted without hiding
the banner itself.

## SDK compatibility

SDK major versions are independent from MTProto/API layers. The host reports
its supported SDK range and capability set. A plugin must declare a compatible
SDK range before install. Unknown major versions are rejected; optional
unknown capabilities are not silently granted.

## Release gates

- Runtime sandbox and broker threat model reviewed.
- Automated tests prove filesystem, network, storage, and capability isolation.
- Fuzz tests cover manifest and package parsing.
- Permission upgrade/revoke/uninstall/rollback tests pass.
- UI failure tests prove plugin crashes cannot block core app navigation.
- Only after these gates may a signed package be considered executable.

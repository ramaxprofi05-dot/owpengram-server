# FemboyGram feature architecture: Extera and profile banners

This work extends the existing OwpenGram-derived Android client and Go server.
It is not a replacement client, a new protocol stack, or a second server.

## FemboyGram Extera

Extera is a distinct advanced client edition built on the same FemboyGram
Android codebase and compatible server APIs. The standard FemboyGram edition
must remain usable without installing extensions.

### User-facing requirements

- A plugin manager with installed, available, update, and disabled states.
- Import a plugin package from local storage and install third-party packages
  from a catalog.
- Create plugins using an in-app developer workflow and a documented SDK.
- Inspect a plugin's publisher, version, target, requested permissions, and
  signature before installation.
- Grant or revoke permissions per plugin; disable and uninstall plugins.
- Support extension points for themes and profile/chat UI customization first.
- Keep extension failures isolated so a broken plugin cannot prevent login,
  open chats, or recover the app's normal UI.

### Security and compatibility rules

The existing server package catalog, signed archive format, publisher trust
store, and review history are the foundation. They are not yet a plugin runtime.
Catalog approval must never be treated as permission to execute code.

Android plugins must run inside a constrained runtime with an explicit
capability bridge. No unrestricted reflection, arbitrary filesystem access,
shell execution, or implicit access to messages, contacts, credentials, or
media. Sensitive capabilities require a clear user grant and revocation path.
Server-target extensions remain WASM-only and must use a separately enforced
capability boundary. Never execute uploaded packages in the admin process.

The plugin SDK must be versioned independently from the Telegram-compatible
MTProto layer. A plugin declares its SDK range and optional UI/API capabilities.
Unknown capabilities fail closed. Package updates must preserve the user's
permission decisions only when the new version requests no additional access;
new permissions require renewed consent.

### Delivery sequence

1. Finish catalog lifecycle and publisher trust/review UX.
2. Define and publish the Android SDK, manifest schema version, and stable
   extension-point interfaces.
3. Add the sandbox/runtime and capability broker with automated isolation tests.
4. Add local import, install/update/rollback, permission review, disable, and
   uninstall flows.
5. Add the in-app plugin editor/templates and third-party discovery catalog.
6. Add theme and profile customization APIs, then expand extension points.

## Profile video banners

A profile banner is a user-selected video displayed as a full-screen visual
surface when another person opens that user's profile. It is public profile
content, not a private story and not a replacement for the avatar.

### Data and media ownership

- Video bytes use the existing document/media upload and object-storage path.
- The profile banner record references the stored document; it does not copy
  media into a new storage subsystem.
- One active banner is associated with each user profile.
- The media reference is registered with the existing media-retention system
  so cleanup cannot delete a document still used by a profile.
- Replacing or removing a banner must update the media reference in the same
  database transaction as the banner row.
- Removing a banner removes only the profile reference. Normal media cleanup
  decides when an otherwise unreferenced document can be purged.

The first server migration adds the durable profile-to-document relationship.
RPC upload/update/delete handlers, public profile projection, and Android
rendering are follow-up implementation steps; the migration alone does not
claim that banners are already visible in clients.

### UX and playback

- Profile owner can upload, replace, preview, and remove the banner.
- Viewers see the banner without needing to be contacts.
- Playback is muted by default, with an explicit sound control.
- The video fills the profile header area using a crop/cover presentation;
  avatar, name, username, bio, badges, and profile actions remain legible.
- Provide a static poster/fallback while the video loads or when playback is
  unavailable, and respect system reduced-motion/data-saving settings.
- Do not autoplay audio. Pause playback when the profile is no longer visible.

### Acceptance criteria

- Upload and replacement enforce ownership: users can modify only their own
  banner.
- A public profile response exposes only the active banner media reference,
  never private upload metadata.
- Deleting a banner clears its media reference and leaves unrelated profile
  photos and messages untouched.
- A banner document remains protected from orphan cleanup while referenced.
- Invalid, oversized, non-video, or inaccessible media is rejected safely.
- Android handles loading, playback failure, offline state, and fallback art.

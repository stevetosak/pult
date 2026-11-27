---
status: accepted
date: 2026-11-27
---

# A Go core behind Kotlin and Compose screens, Android only

Pult is the Operator's main tool for remote access from the phone: a list of
Actions per Machine, each run with one tap, a real terminal for the Actions
that prompt, and room for more Features (files, status, port forwards) and
more Machines. Ready-made apps were tried on paper first and rejected by the
Operator on 2026-11-27: ServerBox and Termius keep snippets in the app, HTTP
Shortcuts needs a command service on the Machine and cannot carry a password
prompt, and KDE Connect returns no output.

**The logic is a Go core; the screens are Kotlin with Jetpack Compose.** The
Core holds the Machines, the SSH sessions (a command with its output streams,
and a terminal with resize), the Runs and their history, and the pins. gomobile
binds it into the app as an Android library. The screens are Kotlin, Compose
and Material 3. Tailscale's Android app is built the same way, with a Go core
under Compose screens, so the split is proven at a larger size.

**The Action file format is one Go package.** The `action` package parses,
validates and hashes an Action, and turns its Parameters into environment
variables. The `pult` command on each Machine and the Core both import it, so
the two sides can never disagree about what an Action is or what its pin
covers.

**The terminal is connectbot/termlib.** It is a Compose component over
libvterm (Apache-2.0, version 0.3.15 of 2026-11-12), maintained with the
ConnectBot app. The Core's terminal session sends the bytes across the Bridge
to the component, and the keys back.

**The Bridge passes simple types.** gomobile carries numbers, strings, byte
slices and interfaces. Structured values cross as JSON, and streams come back
through interfaces that Kotlin implements. The Bridge stays thin: it holds no
logic that a Go test cannot reach, and it turns a Go panic into an error
instead of a native crash.

**Android only.** The Operator's phone is Android. An iOS app would put SwiftUI
screens on the same Core; it is not planned.

**tsnet is why the Core is Go and not Kotlin.** In the first version, the app
reaches the Machines through the phone's Tailscale app. tsnet, which exists only
in Go, lets the app later join the tailnet as its own device: it then works
while another VPN holds Android's single VPN slot, and the tailnet policy can
grant SSH to that device alone instead of to the whole phone. Adding it to a
Kotlin core later would mean rewriting the SSH layer. The Operator writes Go by
preference, and the shared `action` package is the other gain.

## Considered options

- **Kotlin only**, with sshj or Apache MINA SSHD and the same terminal
  component. One language and the simplest build, with ConnectBot as a working
  example; but the Action file format would be defined twice, in Kotlin and in
  Go for the `pult` command, and tsnet would be out of reach.
- **Flutter**, with dartssh2 and xterm.dart. Android, iOS and desktop from one
  code base, and ServerBox shows it works; but Dart is a new language for the
  Operator, and xterm.dart has had no change since 2025-06.
- **Go only**, with Gio or Fyne. One language; but there is no terminal
  component for a phone (fyne-io/terminal has no release), and keyboard and
  input-method support is weaker than Compose's.
- **Rust**, with Tauri mobile or Dioxus. Mobile support is not mature enough
  for an app used every day.

## Consequences

- The build needs the Android SDK, the NDK and gomobile, all pinned; CI binds
  the Core into an Android library before Gradle builds the app.
- Debugging crosses the Bridge: a fault in the Core is found with Go tests
  first, and the Bridge returns errors instead of crashing.
- An SSH key in the Android Keystore, for a Machine without Tailscale SSH,
  signs through a call from the Core back to Kotlin.
- The screens are Kotlin, so the Operator reads Kotlin for every screen change.

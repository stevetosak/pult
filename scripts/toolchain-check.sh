#!/bin/sh
# Compares the Android toolchain on this machine with the pins in
# toolchain.env, which `make toolchain-check` passes in the environment.
# Each mismatch prints the command that installs the pinned version; the exit
# code is 1 when one or more items differ.
set -eu

sdk=${ANDROID_HOME:?ANDROID_HOME is not set}
sdkmanager="$sdk/cmdline-tools/latest/bin/sdkmanager --sdk_root=$sdk"
faults=0

# revision prints the Pkg.Revision of the SDK package installed in $1, or
# nothing when it is not installed. The NDK writes "Pkg.Revision = x", the
# other packages "Pkg.Revision=x".
revision() {
	sed -n 's/^Pkg\.Revision[[:space:]]*=[[:space:]]*//p' "$1/source.properties" 2>/dev/null || true
}

ok() {
	printf 'ok    %-14s %s\n' "$1" "$2"
}

# fail NAME WANT HAVE FIX
fail() {
	printf 'FAIL  %-14s want %s, have %s\n      fix: %s\n' "$1" "$2" "${3:-nothing}" "$4"
	faults=$((faults + 1))
}

# expect NAME WANT HAVE FIX
expect() {
	if [ "$2" = "$3" ]; then ok "$1" "$3"; else fail "$@"; fi
}

expect cmdline-tools "$ANDROID_CMDLINE_TOOLS" "$(revision "$sdk/cmdline-tools/latest")" \
	"unpack the command-line tools $ANDROID_CMDLINE_TOOLS into $sdk/cmdline-tools/latest (README.md)"

# sdkmanager installs a platform by its name only, so the name is the pin and
# the revision is shown.
platform_revision=$(revision "$sdk/platforms/$ANDROID_PLATFORM")
if [ -n "$platform_revision" ]; then
	ok platform "$ANDROID_PLATFORM (revision $platform_revision)"
else
	fail platform "$ANDROID_PLATFORM" "" "$sdkmanager --install \"platforms;$ANDROID_PLATFORM\""
fi

expect ndk "$ANDROID_NDK" "$(revision "$sdk/ndk/$ANDROID_NDK")" \
	"$sdkmanager --install \"ndk;$ANDROID_NDK\""

# go.mod pins gomobile and gobind; this shows the version the build uses.
ok gomobile "$(go list -m -f '{{.Version}}' golang.org/x/mobile) (go.mod)"

# gomobile compiles the Java side with javac; the JDK is not pinned here.
if command -v javac >/dev/null 2>&1; then
	ok javac "$(javac -version 2>&1)"
else
	fail javac "a JDK" "" "install a JDK that has javac"
fi

if [ "$faults" -gt 0 ]; then
	echo "$faults item(s) differ from toolchain.env" >&2
	exit 1
fi

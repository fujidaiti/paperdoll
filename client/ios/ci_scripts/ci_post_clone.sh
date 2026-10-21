#!/bin/sh
#
# Prepares the Xcode Cloud build machine to build the app.
#
# That workflow has to define these environment variables:
#
#   DART_DEFINES_BASE64   base64-encoded dotenv file holding the app's
#                         --dart-define values
#   APP_XCCONFIG_BASE64   base64-encoded Config/App.xcconfig

FLUTTER_PROJECT_DIR="$CI_PROJECT_FILE_PATH/../.."

set -e
brew install jq
cd "$CI_PRIMARY_REPOSITORY_PATH"

##### Create Version.xcconfig #####

VERSION_FILE="$FLUTTER_PROJECT_DIR/ios/version.json"
tool/version.sh vet "$VERSION_FILE"
# Extract version name dropping the 'beta' suffix, which Apple doesn't accept.
VERSION_NAME=$(jq -r '.versionName // empty' "$VERSION_FILE" | sed 's/\.beta$//')

# Flutter/Release.xcconfig loads this xcconfig.
# We ignore the build number from version.json here
# and use the built-in incremental counter instead.
cat > "$FLUTTER_PROJECT_DIR/ios/Config/Version.xcconfig" <<EOF
FLUTTER_BUILD_NAME=$VERSION_NAME
FLUTTER_BUILD_NUMBER=$CI_BUILD_NUMBER
EOF

##### Create dotenv file #####

if [ -z "$DART_DEFINES_BASE64" ]; then
  echo "DART_DEFINES_BASE64 is not defined." >&2
  exit 1
fi

DOT_ENV="${TMPDIR%/}/.env"
echo "$DART_DEFINES_BASE64" | base64 -d > "$DOT_ENV"

##### Create App.xcconfig #####

if [ -z "$APP_XCCONFIG_BASE64" ]; then
  echo "APP_XCCONFIG_BASE64 is not defined." >&2
  exit 1
fi

echo "$APP_XCCONFIG_BASE64" | base64 -d \
  > "$FLUTTER_PROJECT_DIR/ios/Config/App.xcconfig"

##### Install Flutter SDK #####

FLUTTER_VERSION=$(jq -r '.flutter // empty' .fvmrc)
if [ -z "$FLUTTER_VERSION" ]; then
  echo ".fvmrc or flutter version is missing." >&2
  exit 1
fi

git clone https://github.com/flutter/flutter.git --depth 1 -b "$FLUTTER_VERSION" "$HOME/flutter"
export PATH="$PATH:$HOME/flutter/bin"

# On a fresh clone, the first `flutter` invocation downloads the Dart SDK and
# writes curl's progress meter to stdout, which would corrupt the JSON below.
# Run a throwaway command first so that download happens before we capture
# machine-readable output.
flutter --no-version-check --suppress-analytics --version > /dev/null

FLUTTER_VERSION_OUT="${TMPDIR%/}/flutter-version.json"
flutter --no-version-check --suppress-analytics --version --machine > "$FLUTTER_VERSION_OUT"
echo "flutter --version:"
cat "$FLUTTER_VERSION_OUT"

INSTALLED_VERSION=$(jq -r '.flutterVersion // empty' "$FLUTTER_VERSION_OUT")
if [ "$INSTALLED_VERSION" != "$FLUTTER_VERSION" ]; then
    echo "Flutter version mismatch: got $INSTALLED_VERSION, want $FLUTTER_VERSION" >&2
    exit 1
fi

##### Build #####

cd "$FLUTTER_PROJECT_DIR"
# See https://docs.flutter.dev/deployment/cd#xcode-cloud
flutter precache --ios
# TODO(#161): upload the symbol file to Firebase crashlytics.
flutter build ios \
    --release \
    --config-only \
    --obfuscate \
    --split-debug-info="${TMPDIR%/}/ios-symbols" \
    --dart-define-from-file="$DOT_ENV"

exit 0

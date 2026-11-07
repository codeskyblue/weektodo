fastlane documentation
----

# Installation

Make sure you have the latest version of the Xcode command line tools installed:

```sh
xcode-select --install
```

For _fastlane_ installation instructions, see [Installing _fastlane_](https://docs.fastlane.tools/#installing-fastlane)

# Available Actions

## iOS

### ios teams

```sh
[bundle exec] fastlane ios teams
```

List Apple Developer teams for the current Apple ID

### ios setup_signing

```sh
[bundle exec] fastlane ios setup_signing
```

Create an Apple Distribution certificate and an App Store provisioning profile

### ios beta

```sh
[bundle exec] fastlane ios beta
```

Build the signed IPA and upload it to TestFlight

----

This README.md is auto-generated and will be re-generated every time [_fastlane_](https://fastlane.tools) is run.

More information about _fastlane_ can be found on [fastlane.tools](https://fastlane.tools).

The documentation of _fastlane_ can be found on [docs.fastlane.tools](https://docs.fastlane.tools).

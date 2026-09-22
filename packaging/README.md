# Distribution

`make deb` uses Go and `dpkg-deb` (Debian package `dpkg-dev`). It produces an
amd64 package by default; `ARCH=arm64 make deb` cross-compiles ARM64. Set
`VERSION=0.2.1` to override the package and binary version together.

Install the generated file with:

```sh
sudo apt install ./dist/pwprofiler_0.3.0_amd64.deb
```

The application runs as the invoking user, without a background service or
root requirement. `pwprofiler serve` listens only on loopback. No configuration
or personal wordlists are installed by the package.

The GitHub release workflow publishes both `.deb` files and SHA256SUMS for a
pushed `v*` tag. Before tagging, run the checks and review the release content.

## Installing by package name

`apt install pwprofiler` becomes available only after a user configures an APT
repository that contains this package (or the package enters an official distro
repository). GitHub release attachments alone do not constitute an APT repository.

A publisher must choose a repository host and signing key, generate Packages
and Release indexes using Debian's repository tools (for example aptly or
reprepro), sign the repository metadata, and serve it over HTTPS. Publish the
repository URL and public key fingerprint, with a `signed-by` keyring setup for
users. Keep the signing private key out of the repository. This change does not
provision hosting, publish a key, or claim that such a repository exists.

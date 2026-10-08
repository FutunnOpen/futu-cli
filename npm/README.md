# @futunn/futu-cli

npm wrapper for the `futu` CLI.

```bash
npm install -g @futunn/futu-cli
futu version
```

The default GitHub Release source is `https://github.com/FutunnOpen/futu-cli`.
Set `FUTU_CLI_RELEASE_BASE` only when testing a fork or private release source.

```bash
npm install -g @futunn/futu-cli
```

To test a temporary prerelease:

```bash
FUTU_CLI_VERSION=v0.1.0-test.1 npm install -g ./npm
```

# @futunn/futu-cli

npm wrapper for the `futu` CLI.

Requirements:

- Node.js 18 or newer
- npm 9 or newer

```bash
npm install -g github:FutunnOpen/futu-cli --install-links=true
futu version
```

`--install-links=true` is recommended for GitHub installs so npm copies the packed package instead of linking to a temporary git checkout.

The scoped npm registry package is not published yet. After it is published:

```bash
npm install -g @futunn/futu-cli
```

Update a GitHub npm installation by reinstalling from GitHub:

```bash
npm install -g github:FutunnOpen/futu-cli --install-links=true
```

After npm registry publication:

```bash
npm update -g @futunn/futu-cli
```

The default GitHub Release source is `https://github.com/FutunnOpen/futu-cli`.
Set `FUTU_CLI_RELEASE_BASE` only when testing a fork or private release source.

To test a temporary prerelease:

```bash
FUTU_CLI_VERSION=v0.1.0-test.1 npm install -g . --install-links=true
```

Troubleshooting:

```bash
npm config set script-shell /bin/sh
cd ~
npm install -g github:FutunnOpen/futu-cli --install-links=true --foreground-scripts
```

If you use nvm, ensure `which node` and `which npm` point to the same Node installation.

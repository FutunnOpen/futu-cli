# futu CLI 安装指南

本文档只覆盖 `futu` CLI。拆分为独立仓库后，可直接作为项目安装说明使用。

## GitHub Release 地址

默认 GitHub Release 地址已经内置：

```text
https://github.com/FutunnOpen/futu-cli
```

CLI 会自动访问：

```text
{release_base}/releases/latest
{release_base}/releases/download/{version}/{asset}
```

如需验证 fork 或临时私有仓库，可以用 `FUTU_CLI_RELEASE_BASE` 覆盖默认地址。

## macOS / Linux

推荐安装方式：

```bash
curl -fsSL https://raw.githubusercontent.com/FutunnOpen/futu-cli/main/scripts/install.sh | sh
```

在仓库内验证安装脚本：

```bash
sh scripts/install.sh
```

默认安装路径：

```text
~/.futu/bin/futu
```

安装脚本会自动把 `~/.futu/bin` 写入当前用户的 shell profile。首次安装后，重新打开终端，或按脚本提示执行 source 命令后即可直接运行：

```bash
futu version
```

指定安装目录：

```bash
INSTALL_DIR=/usr/local/bin sh scripts/install.sh
```

Linux 默认下载 musl 包。如需 glibc 包：

```bash
FUTU_CLI_LIBC=glibc sh scripts/install.sh
```

安装指定版本（用于测试 prerelease 或临时 tag）：

```bash
FUTU_CLI_VERSION=v0.1.0-test.1 sh scripts/install.sh
```

## Windows

推荐安装方式：

```powershell
iwr -UseBasicParsing https://raw.githubusercontent.com/FutunnOpen/futu-cli/main/scripts/install.ps1 -OutFile install.ps1
powershell -ExecutionPolicy Bypass -File .\install.ps1
```

在仓库内验证指定版本：

```powershell
$env:FUTU_CLI_VERSION="v0.1.0-test.1"
powershell -ExecutionPolicy Bypass -File scripts/install.ps1
```

默认安装路径：

```text
%LOCALAPPDATA%\FutuCLI\bin\futu.exe
```

## npm

npm 包名：

```text
@futunn/futu-cli
```

安装：

```bash
npm install -g github:FutunnOpen/futu-cli
```

发布到 npm registry 后，也可以使用：

```bash
npm install -g @futunn/futu-cli
```

npm wrapper 版本必须和 Git tag 对齐。例如 npm `0.1.0` 下载 GitHub Release `v0.1.0`。

测试本地 npm wrapper 时可指定临时 Release 版本：

```bash
FUTU_CLI_VERSION=v0.1.0-test.1 npm install -g .
```

## 发布链路

发布使用 GitHub Actions 的 `Release` 手动 workflow，只发布正式 release。

版本号可以手动填写，例如 `v0.1.0`。如果不填写，workflow 会读取根目录 `package.json` 的 `version`，自动生成对应 tag，例如 npm 版本 `0.1.0` 会发布 `v0.1.0`。

workflow 会执行：

```text
go test ./...
scripts/build-release.sh <version>
git tag -a <version> <current-commit>
git push origin <version>
gh release create <version> dist/* --verify-tag
```

如果 tag 已经存在，workflow 会校验它是否指向当前提交；如果不是当前提交，会停止发布，避免源码 tag 和二进制 assets 对不上。

验证正式版本下载：

```bash
sh scripts/install.sh
```

## 验证

```bash
futu version
futu update --check
futu config init
futu auth login
futu auth status
```

## 升级

```bash
futu update
futu update --check
futu update --force
futu update --release-notes
```

如果通过 GitHub npm 安装，重新安装：

```bash
npm install -g github:FutunnOpen/futu-cli
```

发布到 npm registry 后，使用：

```bash
npm update -g @futunn/futu-cli
```

## 卸载

macOS / Linux：

```bash
rm -f ~/.futu/bin/futu
```

Windows：

```powershell
Remove-Item "$env:LOCALAPPDATA\FutuCLI\bin\futu.exe" -Force
```

npm：

```bash
npm uninstall -g @futunn/futu-cli
```

如需同时删除本地配置和 token：

```bash
rm -rf ~/.futu
```

# GoLabel

macOS 和 Windows 标签编辑器，界面使用 MyGo Native UI。

需要 Go 1.27.3 或更新版本。在项目目录运行下面的命令。

```sh
go run .
```

## macOS 打包

在 macOS 上运行以下命令，应用包输出到 `build/darwin-<架构>/`。

```sh
go run github.com/egoist/mygo/cmd/mygo build -skip-dmg
```

## Windows 打包

在项目目录运行以下命令，可在 Windows 本机编译，也可在 macOS 上交叉编译。生成 Setup 需要 [NSIS](https://nsis.sourceforge.io/)：Windows 上 MyGo 会在需要时下载；macOS 上先运行 `brew install makensis`。

```sh
go run github.com/egoist/mygo/cmd/mygo build -platform windows/amd64
```

产物在 `build/windows-amd64/`，适用于 Windows 10/11 64 位（Intel / AMD）：

- `golabel-setup-0.1.0-amd64.exe`：安装到当前用户，提供开始菜单入口、卸载和 `.golabel` 文件关联
- `GoLabel.exe`：单文件版，双击运行，无需安装

只需要单文件 EXE 时，在打包命令末尾加上 `-skip-installer`，不需要 NSIS。转发单文件版时，请附上同目录的 `THIRD-PARTY-NOTICES.txt`。

## 使用

左侧添加富文本、PNG/JPEG、Code 128 或二维码。文字属性支持格式、字体、字号、对齐和横排或竖排。拖动元素调整位置，拖动选框右下角调整大小，也可以在右侧输入毫米尺寸。保存为 `.golabel` 后可再次打开，图片会随模板保存。

打印前安装标签打印机的系统驱动，并在 GoLabel 设置实际标签宽高和打印头 DPI。点击「打印」，在系统对话框选择打印机和份数，并在打印机属性中选用与标签匹配的纸型。首次先打一张，测量尺寸并扫描条码。

「导出 PDF」会生成与标签同尺寸的单页文件，也可以用系统 PDF 阅读器按实际大小（100%）试印。

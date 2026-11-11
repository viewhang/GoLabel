# GoLabel

macOS 和 Windows 标签编辑器，界面使用 MyGo Native UI。

需要 Go 1.27.3 或更新版本。在项目目录运行下面的命令。

```sh
go run .
```

## 代码结构

- `main.go`：应用启动入口
- `internal/editor`：编辑器、画布、标签数据、模板和设置
- `internal/i18n`：中英文文案及翻译
- `internal/platform`：各平台的条码生成、PDF 导出和系统外观
- `internal/printing`：打印任务与 macOS、Windows 系统打印通道
- `resources`：应用图标和打包资源

在项目目录运行 `go test ./...` 执行全部测试。

## macOS 打包

在 macOS 上运行以下命令，应用包输出到 `build/darwin-<架构>/`。

```sh
go run github.com/egoist/mygo/cmd/mygo build -skip-dmg
```

默认只编译当前机器架构（Apple 芯片为 `darwin-arm64`）。要让同一份应用同时支持 Apple 芯片和 Intel Mac，改用通用包：

```sh
go run github.com/egoist/mygo/cmd/mygo build -platform darwin/universal -skip-dmg
```

产物在 `build/darwin-universal/GoLabel.app`，可直接拷贝到其他 Mac 使用。

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

新建或打开的标签在同一窗口的标签栏中切换。点击标签栏的「+」可新建标签，橙色圆点标记未保存的修改。

左侧添加富文本、PNG/JPEG、Code 128 或二维码。文字属性支持格式、字体、字号、对齐和横排或竖排。拖动元素调整位置，拖动四角手柄调整大小，也可以在右侧输入毫米尺寸。保存为 `.golabel` 后可再次打开，图片会随模板保存。

顶栏的「模板」打开模板库，内置商品、食品、零售、物流、资产等六种标签样式，选中后可在右侧预览并「使用模板」新建同规格标签。

点击「打开」旁的下拉箭头可打开最近使用的标签文件。点击「保存」旁的下拉箭头，选择「保存为模板」，可将当前标签的尺寸和内容保存到本机模板库的「我的模板」。再次使用该模板时会新建一份可独立编辑的标签。

顶栏的「设置」（macOS 也可以按 ⌘,）里有外观、界面语言、默认尺寸、默认打印分辨率、显示网格和对齐吸附。偏好保存在本机配置目录，macOS 为 `~/Library/Application Support/GoLabel/settings.json`，Windows 为 `%AppData%\GoLabel\settings.json`，下次启动继续生效。设置左侧导航里的「模板库」与顶栏的「模板」是同一页。

外观可选「浅色」「深色」和「跟随系统」，切换后立即生效并保存。跟随系统时启动读取系统外观，运行中系统改换深色或浅色也会在一秒内跟上：macOS 读 `AppleInterfaceStyle`，Windows 读 `AppsUseLightTheme`，Linux 读 `GTK_THEME`。标签纸在两种外观下都保持白色，毕竟它代表实际打印出来的不干胶；只有面板、工具栏和画布底色随主题变化。

深色配色按设计稿的深色画板取色：画布 `#0E1218`、面板 `#151A22`、成组控件与输入框 `#11161E`、卡片 `#171D26`、浮层 `#1B212B`、文字 `#E8EDF5` / `#A7B2C3` / `#7E8A9D`、强调 `#7C9AFF`、实心主按钮 `#3D5AFE`。深浅两套共用同一份 `palette` 结构，只换色值；`colors.Accent` 管文字与描边，`colors.Primary` 管实心按钮和开关，`colors.Paper` 固定白色。

界面语言可选「简体中文」「English」和「跟随系统」（设置 → 通用 → 偏好 → 语言，右侧下拉选择），切换后界面文字、模板库里的模板名称和示例内容、原生菜单都随之改变，并记在配置文件的 `language` 字段里。「跟随系统」使用 MyGo 匹配系统首选语言，无法匹配时使用简体中文。界面文案集中在 `internal/i18n/messages.go`，每条包含中英文版本。

打印前安装标签打印机的系统驱动，并在 GoLabel 设置实际标签宽高和打印头 DPI。点击「打印」，在系统对话框选择打印机和份数。Windows 会将标签宽高和 100% 缩放提交给驱动；如果驱动实际使用的纸张尺寸不匹配，会停止打印并提示在打印机属性中设置相同纸型。macOS 请在打印机属性中选用与标签匹配的纸型。间隙纸或黑标纸还需在驱动中选择对应纸张类型。首次先打一张，测量尺寸并扫描条码。

「导出 PDF」会生成与标签同尺寸的单页文件，也可以用系统 PDF 阅读器按实际大小（100%）试印。

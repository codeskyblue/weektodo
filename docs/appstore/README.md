# WeekTodo iOS 发布指南

这份文档记录把 WeekTodo 发布到 App Store 的完整流程。代码和打包配置已经准备好，真正上传前需要你的 Apple Developer 账号完成签名和 App Store Connect 操作。

## 1. 前置条件

- 已付费加入 Apple Developer Program（个人或公司账号）。
- Xcode 26 或更新版本已登录开发者账号。
- App Store Connect 可以创建 App。
- 本机检测到两个付费团队：`74Z4C6R8NT` 和 `VGBWXXY4P5`。请选择 WeekTodo 要归属的团队，后文的 `<TEAM_ID>` 替换成该团队 ID。

## 2. 创建签名材料

### 2.1 创建 Apple Distribution 证书

在 Xcode 中：

1. Xcode → Settings → Accounts，选择团队。
2. Manage Certificates… → 左下角 `+` → Apple Distribution。
3. 确认 `security find-identity -v -p codesigning` 能看到 `Apple Distribution: ...`。

也可以在 [Certificates, Identifiers & Profiles](https://developer.apple.com/account/resources/certificates/list) 创建。

### 2.2 注册 App ID

在 Developer Portal → Identifiers：

- Bundle ID：`com.uiautodev.weektodo`
- 类型：App
- Capabilities：不需要额外能力（不使用推送、登录、应用内购买等）。

### 2.3 创建 App Store Provisioning Profile

Developer Portal → Profiles → `+` → App Store：

- App ID：`com.uiautodev.weektodo`
- Certificate：选择刚创建的 Apple Distribution 证书
- 下载 `.mobileprovision`，例如 `~/Downloads/WeekTodo_AppStore.mobileprovision`

## 3. 创建 App Store Connect App

App Store Connect → My Apps → `+` → New App：

- Platform：iOS
- Name：`WeekTodo`
- Primary Language：简体中文
- Bundle ID：`com.uiautodev.weektodo`
- SKU：`weektodo-ios`
- User Access：按需选择

然后填写 `docs/appstore/metadata-zh.md` 和 `metadata-en.md` 中的信息。


## 使用 fastlane + App Store Connect API Key（推荐）

旧的 Apple ID 会话已过期，不再使用缓存登录。请用新的付费账号创建 App Store Connect API Key：

1. 用新账号登录 [App Store Connect](https://appstoreconnect.apple.com)。
2. Users and Access → Integrations → App Store Connect API → Team Keys → `+`。
3. 角色选择 **Admin**，生成后下载 `AuthKey_<KEY_ID>.p8`（只能下载一次）。
4. 记录 **Key ID** 和 **Issuer ID**，把 `.p8` 放到：

   ```bash
   mkdir -p ~/.appstoreconnect/private_keys
   mv ~/Downloads/AuthKey_<KEY_ID>.p8 ~/.appstoreconnect/private_keys/
   ```

5. 设置环境变量（不要提交到仓库）：

   ```bash
   export ASC_KEY_ID="<KEY_ID>"
   export ASC_ISSUER_ID="<ISSUER_ID>"
   export ASC_KEY_FILEPATH="$HOME/.appstoreconnect/private_keys/AuthKey_<KEY_ID>.p8"
   export TEAM_ID="<TEAM_ID>"   # 如 API Key 已限定团队，可留空
   ```

6. 创建 Apple Distribution 证书和 App Store Profile：

   ```bash
   fastlane setup_signing
   ```

7. 创建 App Store Connect App 记录（如果还没有）：

   ```bash
   fastlane create_app
   ```

8. 构建并上传到 TestFlight：

   ```bash
   IOS_VERSION=0.3.0 IOS_BUILD=1 fastlane beta
   ```

`fastlane beta` 会自动构建签名 IPA 并调用 `pilot` 上传。上传前也可以先用 `make ios-ipa` + `./scripts/check-ios-ipa.sh` 检查包。

## 4. 构建 App Store 包

先确认前端、Go 测试通过：

```bash
npm --prefix frontend run check
go test ./...
```

构建签名后的 `.ipa`：

```bash
make ios-ipa \
  IOS_SIGN_IDENTITY="Apple Distribution: Shengxiang Sun (<TEAM_ID>)" \
  IOS_PROVISIONING_PROFILE="$HOME/Downloads/WeekTodo_AppStore.mobileprovision" \
  IOS_VERSION=0.3.0 \
  IOS_BUILD=1
```

输出：`mobile/ios/bin/weektodo.ipa`。

- `IOS_VERSION` 是 App Store 上显示的版本号。
- `IOS_BUILD` 每次上传都必须递增，不能重复。
- 打包会自动：
  - 编译并嵌入 AppIcon 和 LaunchScreen
  - 写入 `PrivacyInfo.xcprivacy`
  - 从 provisioning profile 提取并应用 entitlements
  - 使用安全时间戳签名

### 检查包内容

```bash
./scripts/check-ios-ipa.sh mobile/ios/bin/weektodo.ipa
```

会检查 Info.plist 关键字段、图标、启动屏、隐私清单、签名和 provisioning profile。

## 5. 校验并上传

### 方式一：Transporter（推荐）

1. 打开 Transporter App。
2. 登录 Apple ID。
3. 拖入 `mobile/ios/bin/weektodo.ipa`。
4. Deliver。

### 方式二：altool + App Store Connect API Key

在 App Store Connect → Users and Access → Integrations 创建 API Key，下载 `.p8`：

```bash
xcrun altool --validate-app \
  -f mobile/ios/bin/weektodo.ipa \
  -t ios \
  --apiKey "<KEY_ID>" \
  --apiIssuer "<ISSUER_ID>"

xcrun altool --upload-app \
  -f mobile/ios/bin/weektodo.ipa \
  -t ios \
  --apiKey "<KEY_ID>" \
  --apiIssuer "<ISSUER_ID>"
```

不要把 API Key、密码或 `.p8` 文件提交到仓库。

## 6. TestFlight

1. 上传后在 App Store Connect → TestFlight 等待处理完成。
2. 如提示出口合规，选择“不使用非豁免加密”（`ITSAppUsesNonExemptEncryption=false` 已写入）。
3. 添加内部测试员，在真机安装测试：
   - 新建任务、完成、删除、拖动
   - 切换分类、周视图
   - 设置、语言、主题
   - WebDAV 同步
   - 杀进程后数据仍在

## 7. App Store 列表

仓库已包含可直接上传的截图：

- iPhone 6.9 英寸（1320×2868）：`docs/appstore/screenshots/iphone-6.9-*.png`
- iPad 13 英寸（2064×2752）：`docs/appstore/screenshots/ipad-13-*.png`
- 描述、关键词、副标题
- 支持 URL 和隐私政策 URL
- 分类：效率
- 年龄分级：4+
- 价格：按需设置（当前代码没有内购）

本项目提供：

- `docs/appstore/metadata-zh.md`
- `docs/appstore/metadata-en.md`
- `docs/appstore/review-notes.md`
- `docs/privacy-policy.html`
- `docs/support.html`

启用 GitHub Pages 后，可使用：

- 隐私政策：`https://codeskyblue.github.io/weektodo/privacy-policy.html`
- 支持：`https://codeskyblue.github.io/weektodo/support.html`

如果没有启用 Pages，可暂时使用 GitHub 页面 URL：

- `https://github.com/codeskyblue/weektodo/blob/main/docs/privacy-policy.html`
- `https://github.com/codeskyblue/weektodo/blob/main/docs/support.html`

## 8. 隐私标签

App Store Connect → App Privacy：

- 是否收集数据：**不收集**
- 是否跟踪：**否**
- 第三方广告：无
- 分析：无

说明：App 数据保存在本机；AI 润色和 WebDAV 同步只在用户主动配置后连接用户指定的服务，开发者不收集这些数据。隐私政策中已说明。

## 9. 提交审核

- 审核备注使用 `docs/appstore/review-notes.md`。
- 不需要测试账号。
- 说明 App 可完全离线使用，AI/WebDAV 为可选功能。

常见风险：

- **4.2 Minimum Functionality**：App 内含 WebView，请确保审核时能体现完整的离线任务管理、周视图、分类、周报等原生价值，而不是单纯网页包装。
- **2.1 崩溃**：上传前用 TestFlight 在真机跑完整流程。
- **5.1.1 权限**：已添加本地网络用途说明。
- **隐私清单**：已包含 `PrivacyInfo.xcprivacy`。

## 10. 发布

审核通过后可以选择手动发布或自动发布。建议先手动发布，确认 App Store 页面和截图无误后再点发布。

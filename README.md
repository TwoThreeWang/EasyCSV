# EasyCSV

<p align="center">
  <img src="build/appicon.png" width="128" height="128" alt="EasyCSV Logo">
</p>

<p align="center">
  <strong>一个简洁、高效且支持超大文件的 CSV 查看与编辑器。</strong>
</p>

<p align="center">
  <a href="https://github.com/TwoThreeWang/EasyCSV/releases">
    <img src="https://img.shields.io/github/v/release/TwoThreeWang/EasyCSV?style=flat-square" alt="Latest Release">
  </a>
  <a href="https://github.com/TwoThreeWang/EasyCSV/blob/main/LICENSE">
    <img src="https://img.shields.io/github/license/TwoThreeWang/EasyCSV?style=flat-square" alt="License">
  </a>
  <img src="https://img.shields.io/badge/Built%20with-Wails-red?style=flat-square" alt="Built with Wails">
</p>

## ✨ 简介 (Introduction)

**EasyCSV** 是一款基于 [Wails](https://wails.io) 构建的现代化桌面 CSV 编辑工具。它专为处理 CSV 文件而生，无论是只有几行的配置表，还是数百万行的海量数据日志，EasyCSV 都能轻松驾驭。

相比于 Excel 或传统文本编辑器，EasyCSV 更加轻量、启动更快，并且针对大文件读取进行了深度优化，拒绝卡顿。

## 🚀 功能特性 (Features)

*   **⚡ 秒级打开超大文件**：采用流式读取与虚拟滚动技术（Infinite Scrolling），轻松浏览 GB 级别的 CSV 文件，内存占用极低。
*   **🔍 实时极速搜索**：支持全表快速过滤，输入即搜，精准定位数据。
*   **✏️ 便捷编辑体验**：
    *   支持单元格双击编辑。
    *   支持多行选中与批量删除。
    *   新建行与数据追加。
*   **🛡️ 安全防丢机制**：
    *   编辑未保存状态醒目提示。
    *   关闭/切换文件前的二次确认拦截。
*   **🛠️ 实用工具箱**：
    *   自动检测文件编码（UTF-8 / GBK），告别乱码。
    *   内置版本更新检测，随时获取最新功能。

## 📦 下载与安装 (Download)

请访问 [Releases 页面](https://github.com/TwoThreeWang/EasyCSV/releases) 下载适用于您操作系统的最新版本。

目前支持：
*   ✅ Windows (x64)
*   ⬜ macOS (Planned)
*   ⬜ Linux (Planned)

## 🛠️ 开发构建 (Development)

如果您想自己编译或为本项目贡献代码，请按以下步骤操作。

### 环境要求
*   **Go** 1.21+
*   **Node.js** 16+
*   **Wails CLI**: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`

### 本地运行
```bash
# 克隆仓库
git clone https://github.com/TwoThreeWang/EasyCSV.git
cd EasyCSV

# 安装前端依赖
cd frontend
npm install
cd ..

# 启动开发模式 (支持热重载)
wails dev
```

### 编译打包
```bash
# 构建 Windows 版本
wails build
```

## 🚀 发布流程 (Release Workflow)

本项目配置了 GitHub Actions 自动构建工作流。只需简单的 Git 操作即可自动打包发布：

1.  **提交代码**：确保本地修改已全部提交并推送到 `main` 分支。
2.  **打标签**：创建一个以 `v` 开头的标签（如 `v1.0.0`）。
    ```bash
    git tag v1.0.0
    ```
3.  **推送标签**：
    ```bash
    git push origin v1.0.0
    ```
4.  **等待构建**：前往 GitHub 仓库的 **Actions** 页面查看构建进度。构建完成后，Release 页面会自动出现新的安装包，且软件内的“检查更新”功能将能检测到此版本。

## 🏗️ 技术栈 (Tech Stack)

*   **Core**: [Wails v2](https://wails.io) (Go + WebView2)
*   **Frontend**: [Vue 3](https://vuejs.org/) + Vite
*   **Grid Component**: [AG Grid](https://www.ag-grid.com/) (Community Edition)
*   **Icons**: [Lucide Vue](https://lucide.dev/)

## 📄 许可证 (License)

本项目采用 MIT 许可证开源。详情请参阅 [LICENSE](LICENSE) 文件。

## 👨‍💻 作者 (Author)

**WangTwoThree**
*   GitHub: [@TwoThreeWang](https://github.com/TwoThreeWang)

---
*Made with ❤️ by TwoThreeWang*

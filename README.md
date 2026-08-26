## 1.服务器要求

1. 核心数和内存大小：
   - 1核1G
2. 带宽大小：
   - 使用oss对带宽没有要求

> 以下命令基于Debian。

## 2.docker的安装

1. 安装依赖包：

   ```shell
   sudo apt install apt-transport-https ca-certificates curl gnupg-agent software-properties-common
   ```

2. 添加 Docker 的官方 GPG 密钥：

   ```shell
   curl -fsSL https://download.docker.com/linux/debian/gpg | sudo apt-key add -
   ```

3. 设置 Docker 的稳定版仓库：

   ```shell
   sudo add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/debian $(lsb_release -cs) stable"
   ```

4. 更新软件包索引并安装 Docker：

   ```shell
   sudo apt update
   sudo apt install docker-ce docker-ce-cli containerd.io
   ```
5. 验证 Docker 是否安装成功：
   ```shell
   sudo systemctl status docker
   # 启动docker
   systemctl start docker
   # 查看当前版本号，是否启动成功
   docker version
   # 设置开机自启动
   systemctl enable docker
   ```

****

## 3.安装docker-compose

1. 下载 Docker Compose 的二进制文件：
   ```shell
   sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
   ```
2. 添加执行权限：
   ```shell
   sudo chmod +x /usr/local/bin/docker-compose
   ```
3. 创建软链接：
   ```shell
   sudo ln -s /usr/local/bin/docker-compose /usr/bin/docker-compose
   ```
4. 验证 Docker Compose 是否安装成功：
   ```shell
   docker-compose --version
   ```

## 3.打包运行前端项目

1. 如果你的网站没有打算使用https,将下面这一行代码给注释掉

   ```html
   <meta http-equiv="Content-Security-Policy" content="upgrade-insecure-requests" />
   ```

2. 分别到 `web/blog` 和 `web/admin` 下面执行如下命令 (推荐关闭vscode的Eslint,本项目没有遵循Eslint的规范)

   如果下列命令执行报错，可以尝试替换版本

   参考版本：npm版本为：8.3.1    vue-cli的版本为：5.0.6

   ```shell
   npm ci
   npm run build
   ```

3. 构建结果分别位于 `web/blog/dist` 和 `web/admin/dist`。

4. 将前台构建结果复制到 Caddy 静态目录的 `blog` 子目录。

5. 将后台构建结果复制到 Caddy 静态目录的 `admin` 子目录。

****

## 4.隔离前后端联调

联调使用独立的 Compose 项目 `benetnasch-integration`，包含 PostgreSQL、Redis、Meilisearch、MinIO、后端和临时 Caddy。它不会修改现有容器、现有 Caddy 配置或现有数据卷。

隔离联调期间请访问 `http://127.0.0.1:18080`（博客）和 `http://127.0.0.1:18008`（管理端）。主 Caddy 的 `80/8008` 端口属于另一套生产链路，不用于隔离联调。

一键构建、同步、初始化并验收：

```powershell
Copy-Item .env.integration.example .env.integration
pwsh ./scripts/integration-deploy.ps1
```

如需分步执行，端口为 `18080`（博客）、`18008`（管理端）、`17777`（后端）、`17700`（Meili）和 `19000/19001`（MinIO）。

```powershell
pwsh ./scripts/integration-up.ps1
pwsh ./scripts/integration-seed.ps1
pwsh ./scripts/integration-smoke.ps1
```

联调结束后只清理这个临时项目：

```powershell
pwsh ./scripts/integration-down.ps1 -RemoveVolumes
```

## 5.部署

准备：配置项目下的config.yaml文件，配置docs目录下的docker-compose文件

1.docker build本项目为镜像（更具需要更改暴露端口）

2.新建docs目录下docker-compose文件中的挂载目录，并将编译好的前端VUE文件

3.使用项目docs目录下的docker-compose一键部署

4.数据库导入数据库表

5.重启所有容器

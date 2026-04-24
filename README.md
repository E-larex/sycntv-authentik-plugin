# synctv-authentik

`authentik` 的 SyncTV OAuth2 插件。

这个仓库只保存插件源码，需要放到 `synctv` 主仓库中一起编译。

## 如何使用

1. 从本仓库的 Release 页面下载对应平台的插件二进制，并重命名为 `authentik`

2. 准备插件目录和配置文件

```bash
mkdir -p ./data/plugins/oauth2
cp ./authentik ./data/plugins/oauth2/authentik
chmod +x ./data/plugins/oauth2/authentik
```

在 `./data/config.yaml` 中写入：

```yaml
oauth2_plugins:
  - plugin_file: plugins/oauth2/authentik
    args:
      - https://<authentik>
```

这里的 `baseurl` 只放在 `config.yaml` 里。  
其他配置通过 SyncTV 网页后台填写。

3. 使用仓库自带的 [docker-compose.yml](./docker-compose.yml) 启动

```bash
docker compose up -d
```

### 网页后台配置

启动后进入 SyncTV 后台，找到 `oauth2_authentik` 这一组配置，填写：

- `oauth2_authentik_client_id`
- `oauth2_authentik_client_secret`
- `oauth2_authentik_redirect_url`
- `oauth2_authentik_enabled`

回调地址示例：

```text
https://<synctv>/oauth2/callback/authentik
```

### authentik 侧说明

在 authentik 中创建 OAuth2 / OIDC application 时，请保证：

- Client ID 与网页后台中的 `oauth2_authentik_client_id` 一致
- Client Secret 与网页后台中的 `oauth2_authentik_client_secret` 一致
- Redirect URI 与网页后台中的 `oauth2_authentik_redirect_url` 一致

## 自己构建

如果你想自己编译，需要先克隆 `synctv` 主仓库：

```bash
git clone https://github.com/synctv-org/synctv.git
cd synctv
```

再把本仓库放到下面这个位置：

```text
synctv/internal/provider/plugins/authentik
```

例如：

```bash
rm -rf internal/provider/plugins/authentik
git clone https://github.com/E-larex/sycntv-authentik-plugin internal/provider/plugins/authentik
```

然后在 `synctv` 根目录编译：

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
go build -o ./authentik ./internal/provider/plugins/authentik
```

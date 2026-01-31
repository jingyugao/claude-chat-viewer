# ArgoCD GitLab 集成场景

## 场景1：更新镜像标签（最常见）

**目标**：部署已构建的新版本镜像

```bash
# Clone仓库
glab repo clone group/k8s_deployment
cd k8s_deployment

# 修改部署文件
vim manifests/production/deployment.yaml
# 修改: image: registry.example.com/my-service:v1.2.0
# 改为: image: registry.example.com/my-service:v1.2.1

# 提交并推送
git add manifests/production/deployment.yaml
git commit -m "chore: bump my-service to v1.2.1"
git push origin main
```

**结果**：
- GitLab webhook通知ArgoCD
- ArgoCD检测到镜像变化
- 自动开始部署新版本

---

## 场景2：修改副本数（扩容/缩容）

**目标**：在高流量期间增加副本数

```bash
glab repo clone group/infrastructure
cd infrastructure

# 修改副本数
vim manifests/production/deployment.yaml
# 修改: spec.replicas: 2
# 改为: spec.replicas: 5

# 提交推送
git add manifests/production/deployment.yaml
git commit -m "chore: scale production to 5 replicas"
git push origin main
```

**验证**：
```bash
# 查看Pod扩容
kubectl get pods -n production
```

---

## 场景3：更新ConfigMap配置

**目标**：修改应用配置，无需重建镜像

```bash
glab repo clone group/infrastructure
cd infrastructure

# 修改ConfigMap
vim manifests/production/configmap.yaml
# 修改: LOG_LEVEL: "info"
# 改为: LOG_LEVEL: "debug"

# 推送变更
git add manifests/production/configmap.yaml
git commit -m "chore: enable debug logging"
git push origin main
```

**注意**：ConfigMap更新后，Pod需要重启才能生效
```bash
kubectl rollout restart deployment/my-service -n production
```

或在部署文件中添加注解触发重启：
```yaml
spec:
  template:
    metadata:
      annotations:
        config-version: "2"  # 修改这个数字触发滚动更新
```

---

## 场景4：多环境分支管理

**仓库结构**：
```
main (生产)      → image: v1.1.0
develop (测试)   → image: v1.2.0-rc1
```

**工作流**：

1. **在测试环境测试新版本**（develop分支）：
```bash
glab repo clone group/infrastructure
cd infrastructure
git checkout develop

# 修改测试环境镜像
vim manifests/staging/deployment.yaml
# image: v1.2.0-rc1

git add manifests/staging/deployment.yaml
git commit -m "test: v1.2.0-rc1 in staging"
git push origin develop
```

2. **测试通过后合并到生产**（main分支）：
```bash
git checkout main
git merge develop
git push origin main
```

ArgoCD自动同步，生产环境开始部署。

3. **如需回滚**：
```bash
git revert <commit-hash>
git push origin main
```

---

## 场景5：Helm values更新

**目标**：修改Helm chart的values文件

```bash
glab repo clone group/infrastructure
cd infrastructure

# 修改生产环境values
vim helm/values-prod.yaml
# 修改: replicas: 3
# 或修改: image.tag: v1.5.0

# 推送变更
git add helm/values-prod.yaml
git commit -m "chore: update helm values for production"
git push origin main
```

ArgoCD会自动运行 `helm template` 并部署新配置。



## 故障排查

### ArgoCD没有自动同步

1. **检查webhook配置**：
```bash
# GitLab: 项目 → 设置 → Webhooks → 查看delivery日志
```

2. **检查仓库权限**：
```bash
# ArgoCD是否能访问GitLab仓库
argocd repo list
argocd repo get https://gitlab.example.com/group/infrastructure.git
```

3. **查看应用状态**：
```bash
argocd app get my-service
argocd app logs my-service
```

### 镜像未更新

1. **验证镜像是否存在**：
```bash
docker pull registry.example.com/my-service:v1.2.1
```

2. **检查镜像拉取权限**：
```bash
kubectl get secret -n production | grep regcred
```

3. **强制Pod重启**：
```bash
kubectl rollout restart deployment/my-service -n production
```

### 合并冲突

```bash
# 解决本地冲突
git pull origin main
# 手动编辑冲突的文件
git add .
git commit -m "resolve merge conflict"
git push origin main
```
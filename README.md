# GUS – Gitlab User Synchronizer
Автоматическое пред-создание пользователей из **Keycloak** и назначение им прав в **GitLab**.

> **Версии, на которые рассчитано решение**
> * GitLab CE 17.2.x
> * Keycloak 24.0.x
> * Kubernetes ≥ 1.25 (тестировано на 1.31.4)
> * Go 1.24.4

---

## Функционал приложения
**GUS** запускается по расписанию (CronJob) и:

1. Читает список пользователей каждой «тенант-группы» **Keycloak**.
2. Создаёт недостающих пользователей в **GitLab**, чтобы SSO смог «прилепиться» к готовой учётке.
3. Добавляет их в соответствующую подгруппу **GitLab** с ролью Developer (или любой, указанной в конфиге).
4. Работает идемпотентно: повторные запуски не дублируют действия.

---

## Настройки (values.yaml)

```yaml
# ---------- Docker image -----------
image:
  repository: docker.nexus.example.com/gitlab-user-synchronizer
  tag: 1.0.0
  pullPolicy: IfNotPresent

# ---------- Cron schedule ----------
schedule: "0 */2 * * *"

# -------------- ENV ----------------
env:
  TENANT_LIST: "dr,test,ai,durs"

  KEYCLOAK_BASE_URL: "https://keycloak.example.com"
  KEYCLOAK_REALM: "myrealm"
  KEYCLOAK_CLIENT_ID: "my-sync-client"
  KEYCLOAK_CLIENT_SECRET: "supersecret"

  GITLAB_BASE_URL: "https://gitlab.example.com"
  GITLAB_TOKEN: "glpat-xxxxxxxxxxxxxxxx"
  GITLAB_ROOT_GROUP: "instance"

  GITLAB_USER_ROLE: "developer"
  GITLAB_OVERRIDE_ROLE: "false"

# ------- Kubernetes resources -------
resources:
  limits:
    cpu: "200m"
    memory: "256Mi"
  requests:
    cpu: "100m"
    memory: "128Mi"
```
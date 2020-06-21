# db-initializer

This is useful if you want to initialize a bunch of databases in a single RDBMS instance (e.g. single RDS instance). This usecase is common for cost-saving measures in non-production or small to medium production environments.
To create a database in postgres, mysql (or mariadb), cocroachdb, you need to:

### 1. Create `db-initializer` namespace and github pull secret

Here's the template for your convenience, don't forget to replace credentials:

```yaml
---
apiVersion: v1
kind: Namespace
metadata:
  name: db-initializer
---
apiVersion: v1
kind: Secret
metadata:
  name: image-pull-secret-github
  namespace: db-initializer
type: kubernetes.io/dockerconfigjson
stringData:
  .dockerconfigjson: |
    {
      "auths": {
        "docker.pkg.github.com": {
          "username": "your-github-user",
          "password": "your-personal-access-token"
        }
      }
    }
```

### 2. Apply the manifest

```bash
kubectl apply -f https://raw.githubusercontent.com/kirillDanshin/db-initializer/master/deployment/combined.yaml
```

### 3. Create dsn secret

Here's a template:

```yaml
---
apiVersion: v1
kind: Secret
metadata:
  name: some-secret-name
  namespace: some-namespace
type: Opaque
stringData:
  dsn: postgres://postgres:somepassword@hostname:5432/placeholderdb?sslmode=disable
```

Fill in your details, including the namespace and secret name, and move on to the next step when you're ready.

### 4. Try it out

To use it after setup, you need to add annotations to a namespace.

```yaml
---
apiVersion: v1
kind: Namespace
metadata:
  name: example
  annotations:
    # this can be a list separated by comma without spaces between
    dbinit.k8s.danshin.pro/dbNames: test1
    # name of the secret from #3
    dbinit.k8s.danshin.pro/secretName: db-initializer-dsn
    # if secret's namespace is not 'default', you can choose it here
    dbinit.k8s.danshin.pro/secretNamespace: some-namespace
```

# TODO

- [ ] non-rbac deployment
- [ ] support annotations on different CRDs

Found a bug? File a ticket!

If you want to suggest a feature or anything else, please send a PR, it's highly appreciated.

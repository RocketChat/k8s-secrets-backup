# k8s-secrets-backup

### 🤔 What is it?
A generic tool to backup kubernetes secrets, encrypt the backup and upload it to a S3 bucket.

It was designed to run as a cronjob inside our Kubernetes clusters to backup sealed secrets controller's keys, but it can be used to backup any secret, or secrets depending if the env variable SECRET__NAME is set, or SECRET__LABEL_KEY and SECRET__LABEL_VALUE is. If the label key and value is set, then the output is a k8s SecretList.

Another less important note: Age encryption is done to an ASCII-only "armored" encoding, decryption is transparent for the age command.

#### :ballot_box_with_check: Environment variables (required, except if explicity says optional)
| Name                               | example                                        | help                                                                                |
| ---------------------------------- | ---------------------------------------------- | ----------------------------------------------------------------------------------- |
| AGE_PUBLIC_KEY                     | "age435fgañdfgjñdsflgjgadf"                    | Age public key matching your private key for decrypt                                |
| BACKUP_DIR                         | "backups"                                      | Optional: the directory to dump the backed up secret file before uploading it to s3 |
| S3__BUCKET_NAME                    | "bucket-name"                                  | AWS s3 bucket name to upload the backups                                            |
| S3__PATH                           | "path"                                         | AWS S3 path to upload the backups to                                                |
| S3__REGION                         | "us-east-2"                                    | AWS s3 region name                                                                  |
| S3__ACCESS_KEY                     | "sample-access-key"                            | AWS access key that has upload permission on the s3 bucket                          |
| S3__SECRET_KEY                     | "sample-secret-key"                            | AWS access secret that has upload permission on the s3 bucket                       |
| S3__ENDPOINT                       | "http://minio:9000"                            | Optional: S3 endpoint, to support different s3 providers                            |
| S3__USE_PATH_STYLE                 | "true"                                         | Optional: use path style addressing for s3                                          |
| SECRET__NAMESPACE                  | "kube-system"                                  | The namespace where the secret to backup is                                         |
| SECRET__NAME                       | "name-of-secret"                               | Optional: the secret name to backup (provide this or the secrets label and value)   |
| SECRET__LABEL_KEY                  | "sealedsecrets.bitnami.com/sealed-secrets-key" | Optional: secret label key to filter secrets to backup                              |
| SECRET__LABEL_VALUE                | "active"                                       | Optional: secret label value to filter secrets to backup                            |
| CLUSTER__NAME                      | "your-cluster-name"                            | Optional: The name of the cluster (either provide this value or the values below)   |
| CLUSTER__NAME_CONFIG_MAP_NAMESPACE | "kube-system"                                  | Optional: The namespace where the cluster name configmap is                         |
| CLUSTER__NAME_CONFIG_MAP_NAME      | "cluster-info"                                 | Optional: The name of the configmap with the cluster name                           |
| CLUSTER__NAME_CONFIG_MAP_KEY       | "cluster-name"                                 | Optional: The key of the cluster name in the configmap                              |

#### 🧞 Kubernetes manifests (examples)

Backup sealed secrets controller's keys once per month
```
apiVersion: batch/v1beta1
kind: CronJob
metadata:
  name: sealed-secrets-keys-sentinel
  namespace: operations
spec:
  schedule: "0 1 10 * *"  # every month on the 10th
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: sealed-secrets-keys-sentinel
          containers:
          - name: sealed-secrets-keys-sentinel
            image: rocketchat/k8s-secrets-backup
            imagePullPolicy: Always
            env:
            - name: SECRET__NAMESPACE
              value: kube-system
            - name: SECRET__LABEL_KEY
              value: sealedsecrets.bitnami.com/sealed-secrets-key
            - name: SECRET__LABEL_VALUE
              value: active
            - name: S3__BUCKET_NAME
              value: secretsbackups.your.domain
            - name: S3__PATH
              value: sealed_secrets_keys
            - name: S3__REGION
              value: us-east-2
            - name: AGE_PUBLIC_KEY
              value: age435fgañdfgjñdsflgjgadf
            - name: CLUSTER__NAME
              value: my_cluster
            - name: S3__ACCESS_KEY
              valueFrom:
                secretKeyRef:
                  key: awsAccessKeyID
                  name: sealed-secrets-keys-sentinel-secret
            - name: S3__SECRET_KEY
              valueFrom:
                secretKeyRef:
                  key: awsSecretAccessKey
                  name: sealed-secrets-keys-sentinel-secret
            resources:
              limits:
                cpu: "1"
                memory: 300Mi
              requests:
                cpu: "0.2"
                memory: 100Mi
          restartPolicy: OnFailure
  successfulJobsHistoryLimit: 3
  failedJobsHistoryLimit: 1

---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: sealed-secrets-keys-sentinel-kubesystem
  namespace: kube-system
rules:
- apiGroups: [""]
  resources: ["secrets", "configmaps"] # configmaps are needed if you provide the cluster name in a configmap
  verbs: ["list", "get"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: sealed-secrets-keys-sentinel-kubesystem
  namespace: kube-system
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: sealed-secrets-keys-sentinel-kubesystem
subjects:
- kind: ServiceAccount
  name: sealed-secrets-keys-sentinel
  namespace: operations
---
apiVersion: v1
kind: ServiceAccount
metadata:
  namespace: operations
  name: sealed-secrets-keys-sentinel
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: sealed-secrets-keys-sentinel-operations
  namespace: operations
rules:
  - apiGroups: [""]
    resources: ["secrets"]
    resourceNames: ["sealed-secrets-keys-sentinel-secret"]
    verbs: ["list", "get"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: sealed-secrets-keys-sentinel-operations
  namespace: operations
subjects:
  - kind: ServiceAccount
    name: sealed-secrets-keys-sentinel
    namespace: operations
roleRef:
  kind: Role
  name: sealed-secrets-keys-sentinel-operations
  apiGroup: rbac.authorization.k8s.io

---
# only needed if you provide the cluster name in a configmap
apiVersion: v1
kind: ConfigMap
metadata:
  name: cluster-info
  namespace: kube-system
data:
  cluster-name: your-cluster-name
```


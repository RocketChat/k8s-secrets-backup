# k8s-secrets-backup

### 🤔: What is it? 
A generic tool to backup kubernetes secrets, encrypt the backup and upload it to a S3 bucket.

It was designed to run as a cronjob inside our Kubernetes clusters to backup sealed secrets controller's keys, but it can be used to backup any secret, or secrets depending if the env variable SECRET_NAME is set, or LABEL_KEY and LABEL_VALUE is. If a label key and value to filter a set of secrets is set, then the output is a k8s SecretList.

Important note: It assumes a configmap with the k8s cluster name is previously created on the kube-system namespace. More info on Kubernetes manifests examples section.

Another less important note: Age encryption is done to an ASCII-only "armored" encoding, decryption is transparent for the age command.

#### :ballot_box_with_check: Environment variables (required, except if explicity says optional)
| Name                  | Example                              | Help                                                     |
| --------------------- | ------------------------------------ | -------------------------------------------------------- |
SECRET_NAME   | "mongodb-secret"                     | Optional, the secret name to backup
NAMESPACE          | "kube-system" | The namespace where the secret to backup is
LABEL_KEY        | "sealedsecrets.bitnami.com/sealed-secrets-key" | Optional, secret label key to filter secrets to backup
LABEL_VALUE        | "active"                    | Optional, secret label value to filter secrets to backup
BUCKET_NAME             | "secretsbackups.your.domain"                    | AWS s3 bucket name to upload the backups
S3_FOLDER             | "sealed_secrets_keys/"                               |  AWS s3 folder name to upload the backups
S3_REGION              | "us-east-2"                          | AWS s3 region name
AWS_ACCESS_KEY_ID           | "ADSFASDFAF23423"                       | AWS access key that has upload permission on the s3 bucket 
AWS_SECRET_ACCESS_KEY               | "asdASFadfasdfñiouo3Q334" | AWS access secret that has upload permission on the s3 bucket
AGE_PUBLIC_KEY           | "age435fgañdfgjñdsflgjgadf"                            | Age public key matching your private key for decrypt 


#### 🧞: Kubernetes manifests (examples) 

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
            - name: NAMESPACE
              value: kube-system
            - name: LABEL_KEY
              value: sealedsecrets.bitnami.com/sealed-secrets-key
            - name: LABEL_VALUE
              value: active
            - name: BUCKET_NAME
              value: secretsbackups.your.domain
            - name: S3_FOLDER
              value: sealed_secrets_keys/
            - name: S3_REGION
              value: us-east-2
            - name: AGE_PUBLIC_KEY
              value: age435fgañdfgjñdsflgjgadf
            - name: AWS_ACCESS_KEY_ID
              valueFrom:
                secretKeyRef:
                  key: awsAccessKeyID
                  name: sealed-secrets-keys-sentinel-secret
            - name: AWS_SECRET_ACCESS_KEY
              valueFrom:
                secretKeyRef:
                  key: awsSecretAccessKey
                  name: sealed-secrets-keys-sentinel-secret
            command: ["/app/k8s-secrets-backup"]
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
  resources: ["secrets", "configmaps"]
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
apiVersion: v1
kind: ConfigMap
metadata:
  name: cluster-info
  namespace: kube-system
data:
  cluster-name: your-cluster-name
```


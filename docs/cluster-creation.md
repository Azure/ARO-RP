# Cluster Creation Flow

> See also: [cluster-creation.mm](cluster-creation.mm) (FreeMind/Freeplane mindmap with full detail)

## Overview

```mermaid
flowchart TD
    A[Pre-steps] --> B[az aro create - CLI]
    B --> CRED{Credential model?}
    CRED -->|"Service Principal (legacy)"| SP["Create AAD app + SP<br/>Role assignments for SP"]
    CRED -->|"Managed Identity (new)"| MSI["Set user-assigned MSI<br/>Set platform workload identities"]
    SP --> C[ARM routing]
    MSI --> C
    C --> D[RP Frontend - PUT handler]
    D --> DB[(CosmosDB<br/>cluster + operation records)]
    DB --> E[RP Backend - poll and dequeue]
    E --> INSTALL{Install path?}
    INSTALL -->|Hive| HIVE[Hive ClusterDeployment]
    INSTALL -->|Podman| PODMAN[Podman container install]
    HIVE --> F[Phase 1: Bootstrap]
    PODMAN --> F
    F --> G[Phase 2: RemoveBootstrap]
    G --> H[ProvisioningState: Succeeded / Failed]

    style A fill:#e1f5fe
    style B fill:#e1f5fe
    style SP fill:#fff3e0
    style MSI fill:#e1f5fe
    style D fill:#fff3e0
    style DB fill:#f3e5f5
    style E fill:#fff3e0
    style HIVE fill:#e8f5e9
    style PODMAN fill:#e8f5e9
    style F fill:#e8f5e9
    style G fill:#e8f5e9
```

## Pre-steps

```mermaid
flowchart LR
    P[Pre-steps] --> P1[Get Red Hat pull secret - optional]
    P --> P2[Register ARO RP - az provider register]
    P --> P3[Create vnets]
    P --> P4[Create resource group]
    P --> P5["(MSI) Create user-assigned managed identity"]
    P --> P6["(MSI) Create platform workload identities for operators"]
```

## CLI: az aro create

```mermaid
flowchart TD
    CLI["az aro create<br/>(python/az/aro/azext_aro/custom.py)"] --> V1[Static parameter validation]
    V1 --> V2[Validate RP registered to subscription]
    V2 --> V3[Validate subnets]
    V3 --> V4[Dynamic validation]
    V4 --> ID{Identity model?}

    ID -->|Service Principal| SP1[Create AAD application if needed]
    SP1 --> SP2[Create/retrieve service principal]
    SP2 --> BUILD

    ID -->|Managed Identity| MSI1[Set UserAssigned identity]
    MSI1 --> MSI2[Set PlatformWorkloadIdentityProfile]
    MSI2 --> BUILD

    BUILD[Build OpenShiftCluster object] --> RBAC[ensure_resource_permissions - role assignments]
    RBAC --> PUT[PUT to RP and wait for 201]
    PUT --> POLL[Poll async operation record]
    POLL --> OUT[Output cluster state]

    V4 --> V4a[VNet permissions]
    V4 --> V4b[Resource provider permissions]
    V4 --> V4c[Quota validation]
    V4 --> V4d[Disk encryption set]
    V4 --> V4e[Domain validation]
    V4 --> V4f[CIDR range validation]
    V4 --> V4g[Version validation]
    V4 --> V4h[Outbound type - LB vs UDR]
    V4 --> V4i["(MSI) Identity validation"]
```

## RP Frontend: PUT handler

```mermaid
flowchart TD
    FE["PUT handled by RP frontend<br/>(pkg/frontend)"] --> MW[Middleware chain]
    MW --> MW1[Lowercase]
    MW --> MW2[Log]
    MW --> MW3[Metrics]
    MW --> MW4[Panic]
    MW --> MW5[Headers]
    MW --> MW6[Validate - URL params]
    MW --> MW7[Body]
    MW --> MW8[SystemData - ARM metadata]
    MW --> MW9["Authenticated<br/>(MISE or mutual TLS)"]

    MW --> ROUTE["putOrPatchOpenShiftCluster<br/>(pkg/frontend/openshiftcluster_putorpatch.go)"]

    ROUTE --> R1[Validate subscription state]
    R1 --> R2[Unmarshal request body]
    R2 --> R3[ValidateNewCluster]
    R3 --> R3a["staticValidator.Static()"]
    R3 --> R3b["skuValidator.ValidateVMSku()"]
    R3 --> R3c["quotaValidator.ValidateQuota()"]
    R3 --> R3d["providersValidator.ValidateProviders()"]
    R3 --> R4["(MSI) validatePlatformWorkloadIdentities"]
    R4 --> R5[Validate install version supported]
    R5 --> R6["Set ProvisioningState = Creating"]
    R6 --> R7[Allocate monitoring bucket]
    R7 --> R8["(MSI) Store identity URL and tenant ID"]
    R8 --> R9[Set defaults and operator flags]
    R9 --> R10[Create async operation record in CosmosDB]
    R10 --> R11[Create cluster record in CosmosDB]
    R11 --> R12[Return cluster record - excluding secrets]
```

## RP Backend

```mermaid
flowchart TD
    BE["RP Backend - dequeue from CosmosDB<br/>(pkg/backend)"] --> D1[Backends race to dequeue - one wins lease]
    D1 --> D2[Heartbeat process starts - maintains lease]
    D2 --> D3[Load subscription document]
    D3 --> D4["Determine Hive mode<br/>(installViaHive, adoptViaHive, or neither)"]
    D4 --> D5[Create cluster manager]
    D5 --> D6["m.Install(ctx) - multi-phase install"]
```

## Phase 1: Bootstrap

```mermaid
flowchart TD
    P1["Phase 1: Bootstrap<br/>(pkg/cluster/install.go)"] --> IDCHECK{Identity model?}

    IDCHECK -->|MSI| M1[ensureClusterMsiCertificate]
    M1 --> M2[initializeClusterMsiClients]
    M2 --> M3[platformWorkloadIdentityIDs]
    M3 --> VALIDATE

    IDCHECK -->|SP| VALIDATE

    VALIDATE[Dynamic validation] --> V1[validateResources]
    V1 --> V2[validateZones]
    V2 --> IDCHECK2{Identity model?}

    IDCHECK2 -->|MSI| M4[clusterIdentityIDs]
    M4 --> M5[persistPlatformWorkloadIdentityIDs]
    M5 --> INFRA

    IDCHECK2 -->|SP| S1[initializeClusterSPClients]
    S1 --> S2[clusterSPObjectID]
    S2 --> INFRA

    INFRA[Azure infrastructure setup] --> F1[ensurePreconfiguredNSG - if BYO NSG]
    F1 --> F2[ensureACRToken]
    F2 --> F3[ensureInfraID / ensureSSHKey / ensureStorageSuffix]
    F3 --> F4[populateMTUSize]
    F4 --> F5[createDNS]
    F5 --> F6["createOIDC - OIDC provider creation"]
    F6 --> F7[ensureResourceGroup]
    F7 --> F8[ensureServiceEndpoints]
    F8 --> DEPLOY

    DEPLOY[Network and compute] --> N1[setMasterSubnetPolicies]
    N1 --> N2["deployBaseResourceTemplate<br/>(DNS zones, LBs, VMs, NSGs, storage)"]
    N2 --> N3["(MSI) federateIdentityCredentials"]
    N3 --> N4[attachNSGs]
    N4 --> API

    API[API server and networking] --> A1[updateAPIIPEarly]
    A1 --> A2[createOrUpdateRouterIPEarly]
    A2 --> A3[ensureGatewayCreate]
    A3 --> A4[createAPIServerPrivateEndpoint]
    A4 --> CERT[createCertificates - API server and ingress TLS]
    CERT --> INSTALL
```

```mermaid
flowchart TD
    PRE{"Need Hive namespace?<br/>(Hive install or adopt)"} -->|Yes| H0[hiveCreateNamespace]
    PRE -->|No| INSTALL{Install path?}
    H0 --> INSTALL

    INSTALL -->|Hive| H2[runHiveInstaller - create ClusterDeployment]
    H2 --> H3["hiveClusterInstallationComplete<br/>(wait up to 60 min)"]
    H3 --> H4[generateKubeconfigs]
    H4 --> RESET["hiveResetCorrelationData<br/>(if Hive or adopt)"]

    INSTALL -->|Podman| P1a[runPodmanInstaller]
    P1a --> P2a[generateKubeconfigs]
    P2a --> P3a{"Adopt via Hive?"}
    P3a -->|Yes| P4a[hiveEnsureResources]
    P4a --> P5a["hiveClusterDeploymentReady (5 min)"]
    P5a --> RESET
    P3a -->|No| POST[Post-install bootstrap]

    RESET --> POST
    POST --> Q1[ensureBillingRecord]
    Q1 --> Q2[initializeKubernetesClients - cluster now running]
    Q2 --> Q3[initializeOperatorDeployer]
    Q3 --> Q4["apiServersReady (30 min timeout)"]
    Q4 --> Q5[installAROOperator]
    Q5 --> Q6[enableOperatorReconciliation]
    Q6 --> Q7["incrInstallPhase - transition to Phase 2"]
```

## Bootstrap VM (parallel)

```mermaid
flowchart TD
    BVM["Bootstrap VM executes<br/>(bootkube.sh)<br/>Runs in parallel with RP steps"] --> B1[Provides initial apiserver behind ILB]
    B1 --> B2[Bootstrap etcd and wait for stability]
    B2 --> B3[Run release payload image]
    B3 --> B4[Generate cluster assets]
    B4 --> B5[Repeatedly apply assets against running cluster]
    B5 --> B6[Write bootstrap completed configmap]

    B5 --> OPS[Cluster operators start gradually]
    OPS --> O1[machine-api operator starts]
    O1 --> O2[Worker VMs created]
    O2 --> O3[Ingress - depends on workers]
    O3 --> O4[Console - depends on ingress]
```

## Phase 2: RemoveBootstrap

```mermaid
flowchart TD
    P2["Phase 2: RemoveBootstrap<br/>(pkg/cluster/install.go)"] --> INIT2[Initialize clients]
    INIT2 --> RB[Bootstrap removal]

    RB --> R1["removeBootstrap<br/>(delete VM, NIC, disk)"]
    R1 --> R2["removeBootstrapIgnition<br/>(delete unencrypted, keep encrypted graph)"]
    R2 --> HEALTH

    HEALTH[API server and node health] --> H1["apiServersReady (30 min)"]
    H1 --> H2[configureAPIServerCertificate]
    H2 --> H3["apiServersReady (30 min) - recheck"]
    H3 --> H4["minimumWorkerNodesReady (30 min)"]
    H4 --> CONSOLE

    CONSOLE[Console and UI] --> C1["operatorConsoleExists (30 min)"]
    C1 --> C2[updateConsoleBranding - ARO branding]
    C2 --> C3["operatorConsoleReady (20 min)"]
    C3 --> OSCONFIG

    OSCONFIG[OpenShift configuration] --> O1[disableSamples]
    O1 --> O2[disableOperatorHubSources]
    O2 --> O3[disableUpdates - lock version]
    O3 --> O4["clusterVersionReady (30 min)"]
    O4 --> ARO

    ARO[ARO operator stabilization] --> A1["aroDeploymentReady (20 min)"]
    A1 --> A2[updateClusterData]
    A2 --> NET

    NET[Networking and storage] --> N1[configureIngressCertificate]
    N1 --> N2["ingressControllerReady (30 min)"]
    N2 --> N3[configureDefaultStorageClass]
    N3 --> N4[removeAzureFileCSIStorageClass]
    N4 --> FIN

    FIN[Finalization] --> F1[disableOperatorReconciliation]
    F1 --> F2["clusterOperatorsHaveSettled (30 min)"]
    F2 --> F3["finishInstallation<br/>(clear Install field, mark complete)"]
```

# ARO Resource Provider

This document describes how the instances of the Azure Red Hat OpenShift (ARO) Resource Provider and Gateway are deployed in Azure.

## Overview

The ARO Resource Provider (RP) and Gateway are deployed as **Azure Virtual Machine Scale Sets (VMSS)** using ARM templates. This architecture provides high availability, scalability, and version management capabilities.

## Architecture Diagram

### OpenShift Cluster Creation via Hive

```mermaid
%%{init: {'theme':'dark', 'themeVariables': { 'primaryColor': '#2d3748', 'primaryTextColor': '#ffffff', 'primaryBorderColor': '#ffffff', 'lineColor': '#ffffff', 'arrowheadColor': '#ffffff', 'edgeLabelBackground':'#2d3748', 'clusterBkg': '#1a202c', 'clusterBorder': '#ffffff'}}}%%
graph TB
    %% External Layer
    subgraph "External Request"
        UC[User/Client]
        ARM[Azure Resource Manager]
    end

    %% RP Infrastructure
    subgraph "ARO Resource Provider"
        subgraph "RP VMSS Frontend"
            FRONTEND["Frontend Service
            REST API Handler
            Request Validation"]
        end
        
        subgraph "RP VMSS Backend" 
            BACKEND["Backend Service
            Async Processing
            Worker Pool"]
        end
    end

    %% Gateway Infrastructure
    subgraph "ARO Gateway"
        subgraph "Gateway VMSS"
            GATEWAY["Gateway Service
            TCP Proxy
            TLS SNI Routing
            Private Link Connection"]
        end
    end

    %% Hive Integration Layer
    subgraph "Hive Management Layer"
        HIVEMGR["Hive Cluster Manager
        ClusterDeployment Orchestration"]
        SYNCMGR["Hive SyncSet Manager
        Post-Install Configuration"]
    end

    %% Kubernetes Resources
    subgraph "Hive Kubernetes Resources"
        CD["ClusterDeployment
        Declarative Cluster Spec
        Installation Parameters"]
        SS["SyncSet Resources
        Configuration Manifests
        Policy Enforcement"]
        SECRET["Install Secrets
        Pull Secret
        SSH Keys"]
    end

    %% Installation Engine
    subgraph "OpenShift Installation"
        INSTALLER["OpenShift Installer
        openshift-install binary"]
        IGNITION["Ignition Configs
        Bootstrap Configuration
        Node Initialization"]
    end

    %% Azure Infrastructure
    subgraph "Customer Subscription"
        subgraph "Created Infrastructure"
            MASTERS["Master Nodes
            Control Plane
            etcd Cluster"]
            WORKERS["Worker Nodes
            Application Workloads
            Container Runtime"]
            NETWORK["Network Resources
            Load Balancers
            DNS Records"]
            STORAGE["Storage Resources
            Persistent Volumes
            Registry Storage"]
        end
    end

    %% Data Layer
    subgraph "Data Persistence"
        COSMOS["Cosmos DB
        Cluster State
        Operation Status"]
        KV["Key Vault
        Certificates
        Secrets"]
    end

    %% Cluster Creation Flow via Hive
    UC -->|1 CREATE Cluster| ARM
    ARM -->|2 ARM Request| FRONTEND
    FRONTEND -->|3 Validate Request| BACKEND
    BACKEND -->|4 Enable Hive Features| HIVEMGR
    BACKEND -->|4a Configure Gateway| GATEWAY
    
    %% Hive Orchestration
    HIVEMGR -->|5 Create ClusterDeployment| CD
    HIVEMGR -->|6 Store Install Secrets| SECRET
    HIVEMGR -->|7 Trigger Installation| INSTALLER
    
    %% Installation Process
    CD -->|8 Installation Spec| INSTALLER
    SECRET -->|9 Credentials| INSTALLER
    INSTALLER -->|10 Generate Ignition| IGNITION
    INSTALLER -->|11 Provision Infrastructure| MASTERS
    INSTALLER -->|12 Deploy Workers| WORKERS
    INSTALLER -->|13 Create Networks| NETWORK
    INSTALLER -->|14 Setup Storage| STORAGE
    INSTALLER -->|14a Setup Gateway PE| GATEWAY
    
    %% Post-Installation Configuration
    HIVEMGR -->|15 Create SyncSets| SYNCMGR
    SYNCMGR -->|16 Apply Configurations| SS
    SS -->|17 Configure Cluster| MASTERS
    SS -->|18 Apply Policies| WORKERS
    
    %% State Management
    BACKEND -->|19 Update Status| COSMOS
    HIVEMGR -->|20 Store Metadata| COSMOS
    INSTALLER -->|21 Store Certificates| KV
    
    %% Monitoring Integration
    MASTERS -->|22 Report Status| HIVEMGR
    WORKERS -->|23 Health Metrics| HIVEMGR
    HIVEMGR -->|24 Cluster Ready| BACKEND
    BACKEND -->|25 Update ARM Status| FRONTEND
    
    %% Gateway Communication
    MASTERS -->|26 Outbound Traffic| GATEWAY
    WORKERS -->|27 External Access| GATEWAY
    GATEWAY -->|28 Proxy Connections| UC

    %% Styling
    classDef external fill:#4a90e2,stroke:#87ceeb,stroke-width:2px,color:#ffffff
    classDef rp fill:#5dade2,stroke:#85c1e9,stroke-width:2px,color:#ffffff
    classDef gateway fill:#e74c3c,stroke:#ec7063,stroke-width:2px,color:#ffffff
    classDef hive fill:#52b788,stroke:#74c69d,stroke-width:2px,color:#ffffff
    classDef k8s fill:#9b59b6,stroke:#c39bd3,stroke-width:2px,color:#ffffff
    classDef installer fill:#e69138,stroke:#f4a261,stroke-width:2px,color:#ffffff
    classDef infra fill:#27ae60,stroke:#58d68d,stroke-width:2px,color:#ffffff
    classDef data fill:#fff3b0,stroke:#ffe066,stroke-width:2px,color:#000000

    class UC,ARM external
    class FRONTEND,BACKEND rp
    class GATEWAY gateway
    class HIVEMGR,SYNCMGR hive
    class CD,SS,SECRET k8s
    class INSTALLER,IGNITION installer
    class MASTERS,WORKERS,NETWORK,STORAGE infra
    class COSMOS,KV data
```

### OpenShift Cluster Creation Flow via Hive

The diagram shows the complete flow for creating OpenShift clusters through Hive integration:

**1-4. Request Processing:**
- User submits cluster creation request to Azure Resource Manager
- ARM forwards the request to RP Frontend Service
- Frontend validates request and passes to Backend for async processing
- Backend determines cluster should be created via Hive (feature flag enabled)

**5-7. Hive Orchestration Setup:**
- Hive Cluster Manager creates a ClusterDeployment Kubernetes resource
- Install secrets (pull secret, SSH keys) are securely stored
- OpenShift Installer is triggered with ClusterDeployment specification

**8-14. OpenShift Installation Process:**
- ClusterDeployment provides installation parameters to OpenShift Installer
- Installer retrieves credentials and pull secrets
- Ignition configs are generated for node bootstrapping
- Azure infrastructure is provisioned: Master nodes, Worker nodes, Networks, Storage

**15-18. Post-Installation Configuration:**
- Hive Cluster Manager creates SyncSet resources for post-install configuration
- SyncSet Manager applies configuration manifests and policies
- Cluster components are configured according to organizational standards
- Security policies and operational tools are deployed

**19-21. State Management:**
- Backend service updates cluster status in Cosmos DB
- Hive Cluster Manager stores cluster metadata and installation state
- Installer stores certificates and cluster access credentials in Key Vault

**22-25. Monitoring & Completion:**
- Master and Worker nodes report health status back to Hive
- Hive Cluster Manager monitors cluster readiness
- Once cluster is fully operational, status is reported to Backend
- Backend updates ARM with final cluster creation status

**26-28. Gateway Integration:**
- Master nodes route outbound traffic through the Gateway service
- Worker nodes access external resources via Gateway proxy
- Gateway provides secure, controlled connectivity for cluster egress traffic

This Hive-based approach provides declarative cluster management, GitOps integration, and enhanced operational capabilities compared to direct OpenShift installer usage.
